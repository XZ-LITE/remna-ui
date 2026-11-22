package main

import (
	"context"
	"os/exec"
	runtimeOS "runtime"
	"strings"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
	"remna-ui/internal/admin"
	"remna-ui/internal/dns"
	"remna-ui/internal/generator"
	"remna-ui/internal/probe"
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

// ServiceStartup is called by Wails v3 runtime when the service is registered
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	return nil
}

func (a *App) CheckDNS(domain string) types.DNSResult {
	return dns.CheckDNS(domain)
}

func (a *App) CheckIPBlock(host string, port int) types.IPBlockCheckResult {
	return probe.CheckIPBlock(host, port)
}

func (a *App) ProbeVPNTraffic(host string, port int, decoyDomain string, customVless string, server types.SavedServer, sshPassword string) types.VPNProbeResult {
	if server.Host != "" && sshPassword != "" {
		server.Password = sshPassword
	}
	var runner probe.EgressRunner
	if server.Host != "" {
		runner = func() (bool, int, int, string) {
			return admin.RunServerEgressProbe(server)
		}
	}
	return probe.ProbeVPNNodeTraffic(host, port, decoyDomain, customVless, runner)
}

func (a *App) CleanServerDisk(server types.SavedServer, password string) types.CleanDiskResult {
	return admin.CleanServerDisk(server, password)
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
			if app := application.Get(); app != nil {
				app.Event.Emit("deploy:log", map[string]string{
					"status":  status,
					"message": msg,
				})
			}
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
			payload["admin_creation_mode"] = res.AdminCreationMode
			payload["admin_user"] = res.AdminUser
			payload["admin_pass"] = res.AdminPass
			payload["panel_url"] = res.PanelURL
			payload["sub_url"] = res.SubURL
		}
		if app := application.Get(); app != nil {
			app.Event.Emit("deploy:done", payload)
		}
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

func (a *App) ResetCircuitBreaker(serverID string) {
	admin.ResetServerCircuitBreaker(serverID)
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

func (a *App) ToggleStackPerformance(server types.SavedServer, enable bool) types.CommandResult {
	return admin.ToggleStackPerformance(server, enable)
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

func (a *App) OpenBrowserURL(rawURL string) {
	clean := strings.TrimSpace(rawURL)
	if clean == "" {
		return
	}
	if runtimeOS.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", "", clean)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err == nil {
			return
		}
	}
	if app := application.Get(); app != nil {
		_ = app.Browser.OpenURL(clean)
	}
}

func (a *App) SelectDirectory() string {
	app := application.Get()
	if app == nil {
		return ""
	}
	dialog := app.Dialog.OpenFile().
		SetTitle("Выберите папку для сохранения файлов").
		CanChooseDirectories(true).
		CanChooseFiles(false)
	if win := app.Window.Current(); win != nil {
		dialog.AttachToWindow(win)
	}
	dir, err := dialog.PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) SelectSSHKeyFile() string {
	app := application.Get()
	if app == nil {
		return ""
	}
	dialog := app.Dialog.OpenFile().
		SetTitle("Выберите приватный SSH-ключ (id_rsa, id_ed25519, *.pem)").
		AddFilter("SSH Keys (*, *.pem, id_*)", "*;*.pem;id_*")
	if win := app.Window.Current(); win != nil {
		dialog.AttachToWindow(win)
	}
	file, err := dialog.PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return file
}

func (a *App) DownloadBackup(server types.SavedServer, fileName string) types.CommandResult {
	app := application.Get()
	if app == nil {
		return types.CommandResult{Success: false, Error: "Приложение недоступно."}
	}
	dialog := app.Dialog.SaveFile().
		SetMessage("Сохранить резервную копию на диск").
		SetFilename(fileName).
		AddFilter("Gzip Archive (*.tar.gz)", "*.tar.gz")
	if win := app.Window.Current(); win != nil {
		dialog.AttachToWindow(win)
	}
	targetPath, err := dialog.PromptForSingleSelection()
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
