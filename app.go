package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"remna-ui/internal/admin"
	"remna-ui/internal/dns"
	"remna-ui/internal/generator"
	"remna-ui/internal/remnawave"
	"remna-ui/internal/ssh"
	"remna-ui/internal/types"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) CheckDNS(domain string) types.DNSResult {
	return dns.CheckDNS(domain)
}

func (a *App) TestSSH(host string, port int, user, password, keyPath string) types.SSHTestResult {
	return ssh.TestConnection(host, port, user, password, keyPath)
}

func (a *App) GenerateConfig(plan types.DeploymentPlan) types.GenerateResult {
	return generator.WriteOutputFiles(plan, plan.OutputDir)
}

func (a *App) StartDeployment(plan types.DeploymentPlan) {
	go func() {
		res, err := ssh.DeployRemotely(a.ctx, plan, func(status, msg string) {
			runtime.EventsEmit(a.ctx, "deploy:log", map[string]string{
				"status":  status,
				"message": msg,
			})
		})

		success := err == nil
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}

		payload := map[string]interface{}{
			"success": success,
			"error":   errMsg,
		}
		if res != nil {
			payload["admin_user"] = res.AdminUser
			payload["admin_pass"] = res.AdminPass
			payload["panel_url"] = res.PanelURL
			payload["sub_url"] = res.SubURL
		}
		runtime.EventsEmit(a.ctx, "deploy:done", payload)
	}()
}

func (a *App) GetServers() []types.SavedServer {
	return admin.GetSavedServers()
}

func (a *App) SaveServer(server types.SavedServer) bool {
	return admin.SaveServer(server) == nil
}

func (a *App) DeleteServer(id string) bool {
	return admin.DeleteServer(id) == nil
}

func (a *App) GetServerStatus(server types.SavedServer, password string) types.ServerStatusResult {
	return admin.GetServerStatus(server, password)
}

func (a *App) GetContainerLogs(server types.SavedServer, container string) string {
	return admin.GetContainerLogs(server, container)
}

func (a *App) RestartContainer(server types.SavedServer, container string) types.CommandResult {
	return admin.RestartContainer(server, container)
}

func (a *App) RestartAllStack(server types.SavedServer) types.CommandResult {
	return admin.RestartAllStack(server)
}

func (a *App) FixDatabaseSync(server types.SavedServer) types.CommandResult {
	return admin.FixDatabaseSync(server)
}

func (a *App) ConfigureSubscriptionToken(server types.SavedServer, token string) types.CommandResult {
	return admin.ConfigureSubscriptionToken(server, token)
}

func (a *App) AttachNodeSecret(server types.SavedServer, secretKey string) types.CommandResult {
	return admin.AttachNodeSecret(server, secretKey)
}

func (a *App) OptimizeNetworkBBR(server types.SavedServer) types.CommandResult {
	return admin.OptimizeNetworkBBR(server)
}

func (a *App) ConfigureDockerMirrors(server types.SavedServer) types.CommandResult {
	return admin.ConfigureDockerMirrors(server)
}

func (a *App) CreateBackup(server types.SavedServer) types.CommandResult {
	return admin.CreateBackup(server)
}

func (a *App) PruneDockerSystem(server types.SavedServer) types.CommandResult {
	return admin.PruneDockerSystem(server)
}

func (a *App) ResetSuperadmin(server types.SavedServer) types.CommandResult {
	return admin.ResetSuperadmin(server)
}

func (a *App) RenewSSL(server types.SavedServer) types.CommandResult {
	return admin.RenewSSL(server)
}

func (a *App) ToggleAntiAbuse(server types.SavedServer, enable bool) types.CommandResult {
	return admin.ToggleAntiAbuse(server, enable)
}

func (a *App) TestPanelAPI(panelURL, apiToken string) types.PanelInfoResult {
	return remnawave.TestPanelAPI(panelURL, apiToken)
}

func (a *App) CreateNodeInPanel(panelURL, apiToken string, req types.CreateNodeRequest) types.CreateNodeResponse {
	return remnawave.CreateNodeInPanel(panelURL, apiToken, req)
}

func (a *App) GetPanelNodes(panelURL, apiToken string) []types.PanelNodeItem {
	nodes, _ := remnawave.GetNodes(panelURL, apiToken)
	return nodes
}

func (a *App) GetPanelConfigProfiles(panelURL, apiToken string) []types.PanelProfileItem {
	profiles, _ := remnawave.GetConfigProfiles(panelURL, apiToken)
	return profiles
}

func (a *App) GetPanelSquads(panelURL, apiToken string) []types.PanelSquadItem {
	squads, _ := remnawave.GetInternalSquads(panelURL, apiToken)
	return squads
}

func (a *App) ApplyAntiProbingFirewall(server types.SavedServer, panelIP string, clientPorts []int, hideNodePort, stealthDrop, enableBBR bool) types.CommandResult {
	return admin.ApplyAntiProbingFirewall(server, panelIP, clientPorts, hideNodePort, stealthDrop, enableBBR)
}

func (a *App) AuditNodeSecurity(server types.SavedServer, panelIP string) types.NodeSecurityAudit {
	return admin.AuditNodeSecurity(server, panelIP)
}

func (a *App) DeploySelfStealDecoy(server types.SavedServer, domain string) types.CommandResult {
	return admin.DeploySelfStealDecoy(server, domain)
}

func (a *App) OpenBrowserURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func (a *App) SelectDirectory() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите папку для сохранения файлов",
	})
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) SelectSSHKeyFile() string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите приватный SSH-ключ (id_rsa, id_ed25519, *.pem)",
		Filters: []runtime.FileFilter{
			{DisplayName: "SSH Keys (*, *.pem, id_*)", Pattern: "*;*.pem;id_*"},
		},
	})
	if err != nil {
		return ""
	}
	return file
}

func (a *App) DownloadBackup(server types.SavedServer, fileName string) types.CommandResult {
	targetPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Сохранить резервную копию на диск",
		DefaultFilename: fileName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Gzip Archive (*.tar.gz)", Pattern: "*.tar.gz"},
		},
	})
	if err != nil || targetPath == "" {
		return types.CommandResult{Success: false, Error: "Сохранение отменено пользователем."}
	}
	return admin.DownloadBackupStream(server, fileName, targetPath)
}

func (a *App) RestoreBackup(server types.SavedServer, fileName string) types.CommandResult {
	return admin.RestoreBackup(server, fileName)
}

func (a *App) UpdateStackImages(server types.SavedServer) types.CommandResult {
	return admin.UpdateStackImages(server)
}

func (a *App) InstallCloudflareWARP(server types.SavedServer) types.CommandResult {
	return admin.InstallCloudflareWARP(server)
}

func (a *App) ConfigureRealitySelfSteal(server types.SavedServer, fallbackDomain string) types.CommandResult {
	return admin.ConfigureRealitySelfSteal(server, fallbackDomain)
}

func (a *App) AutoFixSubscriptionToken(server types.SavedServer) types.CommandResult {
	return admin.AutoFixSubscriptionToken(server)
}

func (a *App) UninstallStack(server types.SavedServer, purgeData bool) types.CommandResult {
	return admin.UninstallStack(server, purgeData)
}

func (a *App) CheckAppUpdate(repo string) types.AppUpdateInfo {
	return admin.CheckAppUpdate(repo)
}
