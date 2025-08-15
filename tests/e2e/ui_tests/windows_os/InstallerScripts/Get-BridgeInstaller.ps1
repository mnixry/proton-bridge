# Download the Bridge installer, and install it

# Set variables with Bridge's download link and the download path
# to be used later on
$bridgeDownloadURL = $env:BRIDGE_DOWNLOAD_URL
$bridgeDownloadPath = "$env:CI_PROJECT_DIR/tests/e2e/ui_tests/windows_os/InstallerScripts/Bridge-Installer.exe"

# Write the download link of Bridge to use it if manual re-tests are needed
Write-Output $bridgeDownloadURL

# Download the Bridge-Installer.exe file
Invoke-WebRequest -Uri $bridgeDownloadURL -OutFile $bridgeDownloadPath

if (Test-Path -Path $bridgeDownloadPath) {
    Write-Output "Bridge Installer downloaded."
    $file = Get-Item $bridgeDownloadPath | Select-Object Name, Length
    $size = $file.Length
    $sizeMB = "{0:N2}" -f ($size / 1MB)
    Write-Output "File size in MB: $sizeMB"
} else {
    Write-Output "Bridge installer NOT DOWNLOADED"
}
# Install the downloaded Bridge-Installer.exe file
# The installer is passive, meaning no user interaction is needed
# If the user does not have admin rights, it will still show the UAC prompt
# where a user needs to click on "Yes",
# but this will be not needed since the image in the pipeline will be an
# Admin account

# Argument list for passive install
# $argList = "/passive INSTALLSHORTCUT=yes INSTALLDESKTOPSHORTCUT=yes"

# Install Bridge
$process = Start-Process -Wait -ArgumentList "/passive INSTALLSHORTCUT=yes INSTALLDESKTOPSHORTCUT=yes" -PassThru -FilePath $bridgeDownloadPath

# Check exit code of the installation process to confirm installation
if ($process.ExitCode -eq "0") {
    Write-Output "Bridge installed successfully"
} else {
    Write-Error "Bridge not installed successfully!"
    Write-Error "Installer Exit Code: $($process.ExitCode)"
}

# Delete the installer after installation to clean up the space
Remove-Item -Path $bridgeDownloadPath
