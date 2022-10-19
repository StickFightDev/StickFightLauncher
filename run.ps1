cls
# Please use ./build.bat when creating release builds!

function Invoke-Call {
    param (
        [scriptblock]$ScriptBlock,
        [string]$ErrorAction = $ErrorActionPreference
    )
    & @ScriptBlock
    if (($lastexitcode -ne 0) -and $ErrorAction -eq "Stop") {
        exit $lastexitcode
    }
}

Invoke-Call -ScriptBlock { go build -ldflags="-s -w" -o StickFightLauncher.exe } -ErrorAction Stop
.\StickFightLauncher.exe
