<# : batch launcher - Windows runs the lines below, PowerShell treats them as a comment
@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-Expression ([IO.File]::ReadAllText('%~f0'))"
echo.
pause
exit /b
#>
# Instagram automation installer for Windows 10/11.
# Google Drive -> NotebookLM vertical video -> Claude caption -> Instagram Reel,
# orchestrated by a free self-hosted n8n.
#
# Double-click this file. Re-running is safe: it updates the code and keeps your
# .env and n8n data. To sign in to NotebookLM again, delete
# %USERPROFILE%\.notebooklm\profiles\default\storage_state.json and re-run.
#
# Built __BUILD_DATE__ (version __VERSION__). Everything the project needs is
# embedded in this file.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$InstallDir = if ($env:YTA_HOME) { $env:YTA_HOME } else { Join-Path $env:USERPROFILE 'instagram-automation' }
$NotebookLMVersion = '__NOTEBOOKLM_VERSION__'
$Utf8 = New-Object System.Text.UTF8Encoding($false)

function Bold($t) { Write-Host ""; Write-Host $t -ForegroundColor Cyan }
function Info($t) { Write-Host "  $t" }
function Die($t)  { Write-Host ""; Write-Host "Error: $t" -ForegroundColor Red; exit 1 }
function Test-Cmd($name) { [bool](Get-Command $name -ErrorAction SilentlyContinue) }
function Test-Docker { try { docker info *> $null; return $LASTEXITCODE -eq 0 } catch { return $false } }

Bold "Instagram automation setup  (installs into $InstallDir)"

# ------------------------------------------------------------------ Docker
Bold "1/6  Checking Docker"
if (-not (Test-Cmd docker)) {
    $ans = Read-Host "  Docker Desktop is not installed. Install it with winget now? [Y/n]"
    if ($ans -notmatch '^[Nn]') {
        if (-not (Test-Cmd winget)) { Die "winget is missing. Install Docker Desktop from https://www.docker.com/products/docker-desktop/ and run this again." }
        winget install -e --id Docker.DockerDesktop --accept-package-agreements --accept-source-agreements
        Die "Docker Desktop was installed. Restart Windows if asked, start Docker Desktop once, then run this installer again."
    }
    Die "Install Docker Desktop from https://www.docker.com/products/docker-desktop/ , start it once, then run this installer again."
}
if (-not (Test-Docker)) {
    $dd = Join-Path $env:ProgramFiles 'Docker\Docker\Docker Desktop.exe'
    if (Test-Path $dd) { Start-Process $dd }
    Info "Waiting for Docker to start..."
    for ($i = 0; $i -lt 60 -and -not (Test-Docker); $i++) { Start-Sleep 3 }
    if (-not (Test-Docker)) { Die "Docker is installed but not running. Start Docker Desktop and run this installer again." }
}
$hasCompose = try { docker compose version *> $null; $LASTEXITCODE -eq 0 } catch { $false }
if (-not $hasCompose) { Die "This Docker has no 'docker compose'. Update Docker Desktop." }
Info "Docker is running."

# ------------------------------------------------------------------ files
Bold "2/6  Writing project files"
function Put($rel, $b64) {
    $path = Join-Path $InstallDir $rel
    New-Item -ItemType Directory -Force -Path (Split-Path $path) | Out-Null
    [IO.File]::WriteAllBytes($path, [Convert]::FromBase64String($b64))
}
__PAYLOAD__
Info "Files written to $InstallDir"

# ------------------------------------------------------------------ settings
Bold "3/6  Settings (.env)"
$EnvFile = Join-Path $InstallDir '.env'
if (-not (Test-Path $EnvFile)) { Copy-Item (Join-Path $InstallDir '.env.example') $EnvFile }

function Get-EnvValue($key) {
    foreach ($line in [IO.File]::ReadAllLines($EnvFile)) {
        if ($line.StartsWith("$key=")) { return $line.Substring($key.Length + 1) }
    }
    return ''
}
function Set-EnvValue($key, $value) {
    $lines = [Collections.Generic.List[string]]::new([IO.File]::ReadAllLines($EnvFile))
    $idx = $lines.FindIndex({ param($l) $l.StartsWith("$key=") })
    if ($idx -ge 0) { $lines[$idx] = "$key=$value" } else { $lines.Add("$key=$value") }
    [IO.File]::WriteAllText($EnvFile, ($lines -join "`n") + "`n", $Utf8)
}
function New-Secret {
    $bytes = New-Object byte[] 32
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    -join ($bytes | ForEach-Object { $_.ToString('x2') })
}

if (-not (Get-EnvValue 'N8N_ENCRYPTION_KEY')) { Set-EnvValue 'N8N_ENCRYPTION_KEY' (New-Secret) }
if (-not (Get-EnvValue 'BRIDGE_TOKEN')) { Set-EnvValue 'BRIDGE_TOKEN' (New-Secret) }
if (-not (Get-EnvValue 'NOTEBOOKLM_HOME')) { Set-EnvValue 'NOTEBOOKLM_HOME' ((Join-Path $env:USERPROFILE '.notebooklm') -replace '\\', '/') }
if (-not (Get-EnvValue 'ANTHROPIC_API_KEY')) {
    Info "Claude writes the Instagram caption and hashtags."
    Info "Create an API key at https://platform.claude.com/settings/keys"
    $secure = Read-Host "  Paste your Anthropic API key (input hidden)" -AsSecureString
    $key = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
    if (-not $key) { Die "An Anthropic API key is required." }
    Set-EnvValue 'ANTHROPIC_API_KEY' $key.Trim()
}
if (-not (Get-EnvValue 'INSTAGRAM_ACCESS_TOKEN')) {
    Info "The bridge posts Reels with an Instagram access token for a Business or"
    Info "Creator account. README.md, section ""Instagram access token"", shows how"
    Info "to get one from the Meta developer dashboard (about 10 minutes)."
    $secure = Read-Host "  Paste your Instagram access token (input hidden, Enter to add later)" -AsSecureString
    $igtoken = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
    if ($igtoken) { Set-EnvValue 'INSTAGRAM_ACCESS_TOKEN' $igtoken.Trim() }
}
Info "Settings saved in $EnvFile (edit it to change video style, privacy, language...)"

# ------------------------------------------------------------------ NotebookLM login
Bold "4/6  NotebookLM sign-in"
$NlmHome = (Get-EnvValue 'NOTEBOOKLM_HOME') -replace '/', '\'
if (-not (Test-Path (Join-Path $NlmHome 'profiles\default\storage_state.json'))) {
    $env:Path = (Join-Path $env:USERPROFILE '.local\bin') + ';' + $env:Path
    if (-not (Test-Cmd uv)) {
        Info "Installing uv (Python tool installer)..."
        powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://astral.sh/uv/install.ps1 | iex"
        $env:Path = (Join-Path $env:USERPROFILE '.local\bin') + ';' + $env:Path
    }
    uv tool install --force "notebooklm-py[browser]==$NotebookLMVersion"
    if ($LASTEXITCODE -ne 0) { Die "Could not install notebooklm-py." }
    Info "A Microsoft Edge window opens next. Sign in with the Google account you use"
    Info "for NotebookLM; the window closes by itself once you are signed in."
    $env:NOTEBOOKLM_HOME = $NlmHome
    notebooklm login --browser msedge
    notebooklm auth check --test
    if ($LASTEXITCODE -ne 0) { Die "NotebookLM login check failed. Delete $NlmHome\profiles\default\storage_state.json and run this again." }
} else {
    Info "Existing NotebookLM login found ($NlmHome)."
}

# ------------------------------------------------------------------ Claude Code skill
Bold "5/6  Installing the instagram-automation skill for Claude Code"
$SkillDir = Join-Path $env:USERPROFILE '.claude\skills\instagram-automation'
New-Item -ItemType Directory -Force -Path $SkillDir | Out-Null
Copy-Item (Join-Path $InstallDir 'skills\instagram-automation\SKILL.md') (Join-Path $SkillDir 'SKILL.md') -Force
Info "Copied to $SkillDir (the bridge uses its own copy)."

# ------------------------------------------------------------------ start
Bold "6/6  Starting n8n and the bridge"
Set-Location $InstallDir
docker compose up -d --build
if ($LASTEXITCODE -ne 0) { Die "docker compose failed. See the output above." }
Info "Waiting for n8n..."
$up = $false
for ($i = 0; $i -lt 60 -and -not $up; $i++) {
    try { Invoke-WebRequest -UseBasicParsing http://localhost:5680/healthz -TimeoutSec 3 | Out-Null; $up = $true } catch { Start-Sleep 3 }
}
if (-not $up) { Die "n8n did not come up. See: docker compose logs n8n  (in $InstallDir)" }
docker compose exec -T n8n n8n import:workflow --input=/home/node/workflows/instagram-automation.json | Out-Null
Info "Workflow imported."

if (Get-EnvValue 'INSTAGRAM_ACCESS_TOKEN') {
    Info "Checking the Instagram token..."
    Start-Sleep 5
    try {
        $check = Invoke-RestMethod -Uri http://localhost:8766/instagram/check -Headers @{ 'X-Bridge-Token' = (Get-EnvValue 'BRIDGE_TOKEN') } -TimeoutSec 30
        Info ("Instagram OK: @" + $check.account.username)
    } catch {
        Info "WARNING: Instagram check failed ($($_.Exception.Message)). Fix INSTAGRAM_ACCESS_TOKEN in .env, then: docker compose up -d"
    }
} else {
    Info "WARNING: no Instagram token yet. Add INSTAGRAM_ACCESS_TOKEN to $EnvFile, then: docker compose up -d"
}

Bold "Done. Finish these steps in n8n (http://localhost:5680):"
@'
  1. Create your n8n owner account (stored only on this computer).
  2. Open the workflow "Instagram automation: Drive -> NotebookLM -> Claude -> Instagram Reels".
  3. In "New file in Drive folder": create a Google Drive OAuth2 credential and
     pick the folder to watch. Use the same credential in "Download file".
     (Needs a Google Cloud OAuth client with redirect URI
     http://localhost:5680/rest/oauth2-credential/callback - see README.md.)
  4. Toggle the workflow to Active / Publish it.
  Drop a PDF, Doc or text file into the folder. About 10-30 minutes later the
  Reel is waiting for your approval at http://localhost:8766 - nothing is
  posted to Instagram until you click "Approve & post".
'@ | Write-Host
Info "Guide: $InstallDir\README.md     Logs: cd $InstallDir; docker compose logs -f"
try { Start-Process 'http://localhost:5680'; Start-Process 'http://localhost:8766' } catch { }
