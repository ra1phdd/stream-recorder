package main

import (
	"flag"
	"fmt"
	"log"
	"runtime/debug"
	"stream-recorder/internal/config"
	"stream-recorder/internal/recorder"
	"stream-recorder/pkg/logger"
	"strings"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Перехваченная паника:", r)
			fmt.Println("Стек вызовов:")
			fmt.Printf("%s", debug.Stack())
		}
	}()
	// Инициализация логгепа
	logger.Init("debug")

	configEnv, err := config.NewEnvConfig()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	configPath := flag.String("config", "config.json", "Файл конфигурации")

	// Парсинг флагов
	flag.Parse()

	// Разделяем значения по запятой
	configPathStr := strings.Split(*configPath, ",")
	var configPathList []string
	for _, pathStr := range configPathStr {
		if pathStr != "" {
			configPathList = append(configPathList, pathStr)
		}
	}

	for _, path := range configPathList {
		go func(path string) {
			// Подгрузка конфигурации
			configJSON, err := config.NewJsonConfig(path)
			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			recorder.Init(configEnv, configJSON)
		}(path)
	}

	select {}
}
