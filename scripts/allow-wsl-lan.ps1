# 让同一局域网的手机能访问 WSL 里的开发后端（WSL2 Mirrored 网络模式）。
# 背景：Windows 防火墙关掉不等于放行 —— WSL 的入站由 Hyper-V 防火墙管，
#       默认 NotConfigured（= 拦），所以手机连 http://PC_IP:8080 会超时。
# 用法（需管理员，会弹 UAC）：
#   powershell -NoProfile -ExecutionPolicy Bypass -File allow-wsl-lan.ps1
#   可选端口：powershell ... -File allow-wsl-lan.ps1 -Ports 8080,5174
# 撤销：
#   powershell ... -File allow-wsl-lan.ps1 -Remove
param(
    [int[]] $Ports = @(8080),
    [switch] $Remove
)

# WSL 在 Hyper-V 防火墙里的固定作用域 GUID（与 "WSL" 虚拟机创建者对应）
$WslScope = '{40E0AC32-46A5-438A-A0B2-2B479E8F2E90}'
$out = Join-Path $env:TEMP 'anmo-fw-result.txt'
$log = @()

try {
    $existing = Get-NetFirewallHyperVRule -VMCreatorId $WslScope -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like 'Anmo dev WSL LAN*' }

    if ($Remove) {
        $existing | Remove-NetFirewallHyperVRule
        $log += "已删除规则：$($existing.DisplayName -join '; ')"
    } else {
        $existing | Remove-NetFirewallHyperVRule -ErrorAction SilentlyContinue
        foreach ($p in $Ports) {
            $name = "anmo-wsl-lan-$p"
            New-NetFirewallHyperVRule `
                -Name $name `
                -DisplayName "Anmo dev WSL LAN inbound TCP $p" `
                -Direction Inbound `
                -Action Allow `
                -Protocol TCP `
                -LocalPorts $p `
                -RemoteAddresses 'LocalSubnet' `
                -VMCreatorId $WslScope `
                -PolicyStore ActiveStore | Out-Null
            $log += "已放行 TCP $p（仅来自本机所在子网）"
        }
    }

    Get-NetFirewallHyperVRule -VMCreatorId $WslScope -ErrorAction SilentlyContinue |
        Select-Object DisplayName, Direction, Action, Enabled |
        ForEach-Object { $log += "rule> $($_.DisplayName) / $($_.Direction) / $($_.Action) / $($_.Enabled)" }

    # 本机自测一次（若 LAN IP 通了，手机基本就通了）
    $lanIp = (Get-NetIPAddress -AddressFamily IPv4 |
        Where-Object { $_.IPAddress -like '192.168.*' -and $_.PrefixOrigin -ne 'WellKnown' } |
        Select-Object -First 1).IPAddress
    if ($lanIp -and $Ports.Count -gt 0) {
        try {
            $r = Invoke-WebRequest -UseBasicParsing -TimeoutSec 4 "http://${lanIp}:$($Ports[0])/healthz"
            $log += "自测 http://${lanIp}:$($Ports[0])/healthz -> $($r.StatusCode)"
        } catch {
            $log += "自测 http://${lanIp}:$($Ports[0])/healthz 失败：$($_.Exception.Message)"
        }
    }
    $log += 'OK'
} catch {
    $log += "ERROR: $($_.Exception.Message)"
}

$log | Set-Content -Path $out -Encoding UTF8
Write-Output ($log -join "`n")
