package main

import (
	"bufio"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/database"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/storage"
	"github.com/TimonKK/inmemory-db/internal/database/storage/engine"
	"go.uber.org/zap"
	"os"
	"strings"
)

func main() {
	logger, _ := zap.NewProduction()

	computeInstance := compute.NewCompute(logger)
	engineInstance := engine.NewMemoryEngine()
	storageInstance := storage.NewStorage(engineInstance, logger)

	db := database.NewDatabase(computeInstance, storageInstance, logger)

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

		response, err := db.ExecQuery(trimmedInput)
		if err != nil {
			logger.Error("failed to exec query", zap.Error(err))
		}

		fmt.Println(response)
	}
}
