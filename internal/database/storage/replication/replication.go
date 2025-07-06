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
	r.logger.Info("start sync")
	defer r.logger.Info("end sync")
	localWals, err := r.cm.GetWalFiles()
	if err != nil {
		return err
	}

	// make TCP request to master
	cfg := config.ClientNetworkConfig{
		Address:     r.config.MasterAddress,
		IdleTimeout: 1 * time.Minute,
	}

	client, err := network.NewTCPClient(&cfg, r.logger)
	if err != nil {
		return err
	}

	// получить список wal-сегментов
	res, err := client.Send(string(compute.ReplicationCommandId))
	if err != nil {
		return err
	}

	remoteWals := strings.Split(res, ",")
	r.logger.Info("Got wals from master", zap.String("remoteWals", res))

	// если данные синхроны
	if slices.Equal(localWals, remoteWals) {
		r.logger.Info("all sync!")
		return nil
	}

	// если отличие на один сегмент - скачать только его
	if slices.Equal(localWals, remoteWals[:len(remoteWals)-1]) {
		lastWalName := remoteWals[len(remoteWals)-1]
		data, err := client.Send(fmt.Sprintf("%s %s", string(compute.ReplicationCommandId), lastWalName))
		if err != nil {
			return err
		}

		if err = r.cm.Save(lastWalName, strings.Split(data, ",")); err != nil {
			return err
		}

		r.logger.Info("sync complete", zap.String("wal", lastWalName))

		return nil
	}

	// Если дошли сюда - отличаются больше чем один файл - нужно перекачать все wal-файлы с мастера
	r.logger.Info("slave too old, start full sync!")
	if err := r.cm.Reset(); err != nil {
		return err
	}

	for _, remoteWal := range remoteWals {
		data, err := client.Send(fmt.Sprintf("%s %s", string(compute.ReplicationCommandId), remoteWal))
		if err != nil {
			return err
		}

		err = r.cm.Save(remoteWal, strings.Split(data, ","))
		if err != nil {
			return err
		}

		r.logger.Info("sync complete", zap.String("wal", remoteWal))
	}

	r.logger.Info("full sync complete")

	return nil
}

func (r *Replication) Start(ctx context.Context) error {
	if r.config.ReplicaType == ReplicaTypeSlave {
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

		r.logger.Info("Replication query", zap.String("query", queryStr))

		if query.CommandId() != compute.ReplicationCommandId {
			return "", fmt.Errorf("only replication command expected, but got %s", queryStr)
		}

		wals, err := r.cm.GetWalFiles()
		if err != nil {
			return "", err
		}

		// get wal list
		walFile := query.Key()
		if walFile == "" {
			return strings.Join(wals, ","), nil
		}

		// get specific wal file data
		if !slices.Contains(wals, walFile) {
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
