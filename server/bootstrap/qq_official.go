package bootstrap

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"context"
)

// Start the managed loop even without credentials so console setup takes effect
// without restarting the server. Construction never calls platform APIs.
func InitQQOfficial() {
	runtime := service.NewQQOfficialRuntime(global.DB, global.Config.JWT.Secret)
	service.SetDefaultQQOfficialRuntime(runtime)
	manager := global.EnsureBackgroundTaskManager()
	manager.Go("qq_official_dispatch", func(ctx context.Context) error {
		return runtime.Run(ctx, global.CurrentLogger())
	})
	manager.Go("qq_official_gateway", runtime.RunGateway)
}
