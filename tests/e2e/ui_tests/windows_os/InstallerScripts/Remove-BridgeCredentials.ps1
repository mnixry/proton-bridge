<#
PowerShell script for removing Bridge credentials from
Microsoft Credentials manager
#>

$Bridge = Get-Process "bridge" -ErrorAction SilentlyContinue
$CredentialsData = @((cmdkey /listall | Where-Object{$_ -like "*LegacyGeneric:target=protonmail*"}).replace("Target: ",""))


function Remove-BridgeCredentials {
    # Delete the entries in the credential manager

    for($i=0; $i -le ($CredentialsData.Count -1); $i++){
        [string]$DeleteData = $CredentialsData[$i].trim()
        cmdkey /delete:$DeleteData
    }
}

function Stop-PMBridge {
    # Stop the `bridge` process to completely quit Bridge

    if ($Bridge){
        
        $Bridge | Stop-Process -Force

    }
}

function Invoke-Functions{
    Stop-PMBridge
    Remove-BridgeCredentials
}

Invoke-Functions
