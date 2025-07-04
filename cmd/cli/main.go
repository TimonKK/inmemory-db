package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database/network"
	"go.uber.org/zap"
	"io"
	"os"
	"strings"
)

func main() {
	address := flag.String("address", "127.0.0.1:3223", "server address (required), e.g. 127.0.0.1:3223")
	idleTimeout := flag.Duration("timeout", 0, "timeout for server connection, e.g. 5s, 1m")
	flag.Parse()

	clientNetworkConfig := config.ClientNetworkConfig{
		Address:     *address,
		IdleTimeout: *idleTimeout,
	}

	logger, _ := zap.NewProduction()
	logger.Info("Loading clientNetworkConfig", zap.Any("clientNetworkConfig", clientNetworkConfig))

	client, err := network.NewTCPClient(&clientNetworkConfig, logger)
	if err != nil {
		logger.Fatal("Error connecting", zap.Error(err))
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(":)")
		input, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("failed to read", zap.Error(err))
		}

		trimmedInput := strings.TrimSpace(input)

		if strings.ToLower(trimmedInput) == "exit" {
			fmt.Println("bye...")
			break
		}

		response, err := client.Send(trimmedInput)
		if err != nil {
			if err == io.EOF {
				logger.Fatal("client connection closed", zap.Error(err))
				return
			}

			logger.Error("failed to exec query", zap.Error(err))
		}

		fmt.Println(response)
	}
}
