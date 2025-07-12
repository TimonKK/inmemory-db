package replication

import (
	"context"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/network"
	"github.com/TimonKK/inmemory-db/internal/database/storage/wal"
	"go.uber.org/zap"
	"slices"
	"strings"
	"time"
)

type Replication struct {
	tcpServer *network.TCPServer
	tcpClient *network.TCPClient
	cm        *wal.ChunkManager
	compute   database.Compute
	config    *config.ReplicationConfig
	logger    *zap.Logger
}

func NewReplication(cm *wal.ChunkManager, compute database.Compute, cfg *config.ReplicationConfig, logger *zap.Logger) *Replication {
	return &Replication{
		cm:      cm,
		compute: compute,
		config:  cfg,
		logger:  logger,
	}
}

func (r *Replication) startBackgroundWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.config.SyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.logger.Error("context canceled", zap.Error(ctx.Err()))
				return
			case <-ticker.C:
				if err := r.sync(); err != nil {
					r.logger.Error("sync failed", zap.Error(err))
				}
			}
		}
	}()
}

func (r *Replication) sync() error {
	r.logger.Debug("start sync")
	defer r.logger.Debug("end sync")

	localWals, err := r.cm.GetWalFiles()
	if err != nil {
		return err
	}

	// получить список wal-сегментов
	res, err := r.tcpClient.Send(string(compute.ReplicationCommandId))
	if err != nil {
		return err
	}

	r.logger.Debug("Got wals from master", zap.String("remoteWals", res))
	remoteWals := strings.Split(res, ",")

	// если данные синхроны
	if slices.Equal(localWals, remoteWals) {
		r.logger.Debug("all sync!")
		return nil
	}

	// если отличие на один сегмент - скачать только его
	if slices.Equal(localWals, remoteWals[:len(remoteWals)-1]) {
		return r.downloadWal(remoteWals[len(remoteWals)-1])
	}

	// Если дошли сюда - отличаются больше чем один файл - нужно перекачать все wal-файлы с мастера
	r.logger.Debug("slave too old, start full sync!")
	if err := r.cm.Reset(); err != nil {
		return err
	}

	return r.downloadAllWals(remoteWals)
}

func (r *Replication) replicationExec(query compute.Query) (string, error) {
	localWalFiles, err := r.cm.GetWalFiles()
	if err != nil {
		return "", err
	}

	// get wal list
	walFile := query.Key()
	if walFile == "" {
		return strings.Join(localWalFiles, ","), nil
	}

	// get specific wal file data
	if !slices.Contains(localWalFiles, walFile) {
		return "", fmt.Errorf("replication file %s not found", walFile)
	}

	var res strings.Builder
	for line, err := range r.cm.ChunkLines(walFile) {
		if err != nil {
			return "", err
		}

		res.WriteString(line + ",")
	}

	return res.String(), nil
}

func (r *Replication) downloadWal(walName string) error {
	data, err := r.tcpClient.Send(fmt.Sprintf("%s %s", string(compute.ReplicationCommandId), walName))
	if err != nil {
		return err
	}

	if err = r.cm.Save(walName, strings.Split(data, ",")); err != nil {
		return err
	}

	r.logger.Debug("sync complete", zap.String("wal", walName))

	return nil
}

func (r *Replication) downloadAllWals(remoteWals []string) error {
	for _, remoteWal := range remoteWals {
		if err := r.downloadWal(remoteWal); err != nil {
			return err
		}

		r.logger.Debug("sync complete", zap.String("wal", remoteWal))
	}

	r.logger.Debug("full sync complete")

	return nil
}

func (r *Replication) Start(ctx context.Context) error {
	if r.config.ReplicaType == ReplicaTypeSlave {
		tcpClient, err := network.NewTCPClient(&config.ClientNetworkConfig{
			Address:     r.config.MasterAddress,
			IdleTimeout: 1 * time.Minute,
		}, r.logger)
		if err != nil {
			return err
		}

		r.tcpClient = tcpClient

		r.startBackgroundWorker(ctx)
		return nil
	}

	replicationCfg := config.NetworkConfig{
		Address:        r.config.MasterAddress,
		MaxConnections: r.config.MaxReplicasNumber,
	}
	tcpServer, err := network.NewTCPServer(replicationCfg, r.logger)
	if err != nil {
		r.logger.Fatal("Failed to init replication  server", zap.Error(err))
	}

	r.tcpServer = tcpServer

	if err := r.tcpServer.Start(); err != nil {
		return err
	}

	r.logger.Info("Replication started", zap.String("address", r.config.MasterAddress))

	r.tcpServer.HandleConnect(ctx, func(ctx context.Context, queryStr string) (string, error) {
		query, err := r.compute.ParseQuery(queryStr)
		if err != nil {
			return "", err
		}

		r.logger.Debug("Replication query", zap.String("query", queryStr))

		if query.CommandId() != compute.ReplicationCommandId {
			return "", fmt.Errorf("only replication command expected, but got %s", queryStr)
		}

		return r.replicationExec(query)
	})

	return nil
}

func (r *Replication) ValidateQuery(query string) error {
	if r.config.ReplicaType == ReplicaTypeMaster {
		return nil
	}

	if query == string(compute.GetCommandId) {
		return nil
	}

	return fmt.Errorf("for slave allowed only GET queries")
}

func (r *Replication) Stop() error {
	if r.tcpServer == nil {
		return nil
	}

	return r.tcpServer.Shutdown()
}
