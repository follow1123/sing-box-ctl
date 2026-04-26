$binPath = $env:SINGBOX_SERVICE_BINARY
$workingDir = $env:SINGBOX_SERVICE_WORKING_DIR
$configPath = $env:SINGBOX_SERVICE_CONFIG_PATH
$action = $env:SINGBOX_SERVICE_ACTION

$highPermission = $env:SINGBOX_SERVICE_HIGH_PERMISSION

function ServiceStart {
	$proc = ServiceProcess
	if ($proc) {
		return
	}
	if ($highPermission) {
		Start-Process -FilePath $binPath -ArgumentList "-D $workingDir","-c $configPath", "run" -WindowStyle Hidden -Verb RunAs
	} else {
		Start-Process -FilePath $binPath -ArgumentList "-D $workingDir","-c $configPath", "run" -WindowStyle Hidden
	}
}

function ServiceStop {
	$proc = ServiceProcess
	if ($proc) {
		$proc | Stop-Process
		if (-not $?) {
			$proc | Stop-Process -Force
			if (-not $?) {
				$psPath = Join-Path $PSHome "powershell.exe"
				$stopCmd = 'Stop-Process -Name sing-box -Force'
				Start-Process -FilePath $psPath -ArgumentList "-NoProfile","-Command $stopCmd" -Verb RunAs -WindowStyle Hidden
			}
		}
		TurnOffSystemProxy
	}
}

function ServiceProcess {
	Get-Process -Name sing-box
}

function TurnOffSystemProxy {
	$proxyStatus = Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyEnable
	if ($proxyStatus.ProxyEnable -eq 1) {
		Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyEnable -Value 0
	}
}

switch ($action.ToLower()) {
	"start" {
		ServiceStart
		Start-Sleep -Milliseconds 500
		$proc = ServiceProcess
		if ($proc) {
			exit 0
		} else {
			exit 1
		}
	}
	"stop" {
		ServiceStop
		Start-Sleep -Milliseconds 500
		$proc = ServiceProcess
		if ($proc) {
			exit 1
		} else {
			exit 0
		}
	}
	"restart" {
		ServiceStop
		Start-Sleep -Milliseconds 500
		ServiceStart
		Start-Sleep -Milliseconds 500
		$proc = ServiceProcess
		if ($proc) {
			exit 0
		} else {
			exit 1
		}
	}
	"is_running" {
		$proc = ServiceProcess
		if ($proc) {
			Write-Output "true"
		} else {
			Write-Output "false"
		}
	}
}
