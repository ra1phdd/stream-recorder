package tasks

import (
	"context"
	"stream-recorder/config"
	"stream-recorder/internal/app/services/recorder"
	"sync"
)

type TaskInfo struct {
	Platform string
	Username string
	Ctx      context.Context
	Cancel   context.CancelFunc
}

// Глобальная мапа для хранения рабочих горутин
var (
	tasks     = make(map[string]TaskInfo)
	tasksLock sync.Mutex
)

func StartTask(configEnv *config.ConfigurationEnv, username string, platform string, quality string) {
	ctx, cancel := context.WithCancel(context.Background())

	tasksLock.Lock()
	tasks[platform+"_"+username] = TaskInfo{
		Platform: platform,
		Username: username,
		Ctx:      ctx,
		Cancel:   cancel,
	}
	tasksLock.Unlock()

	go recorder.Init(ctx, configEnv, username, platform, quality)
}

func CutTask(configEnv *config.ConfigurationEnv, username string, platform string, quality string) {
	ctx, cancel := context.WithCancel(context.Background())

	go recorder.Init(ctx, configEnv, username, platform, quality)

	tasksLock.Lock()
	tasks[platform+"_"+username] = TaskInfo{
		Platform: platform,
		Username: username,
		Ctx:      ctx,
		Cancel:   cancel,
	}

	if task, ok := tasks[platform+"_"+username]; ok {
		task.Cancel()
		delete(tasks, platform+"_"+username)
	}
	tasksLock.Unlock()
}

func StopTask(username string, platform string) {
	tasksLock.Lock()
	if task, ok := tasks[platform+"_"+username]; ok {
		task.Cancel()
		delete(tasks, platform+"_"+username)
	}
	tasksLock.Unlock()
}
