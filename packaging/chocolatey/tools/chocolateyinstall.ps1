$ErrorActionPreference = 'Stop'

$packageArgs = @{
  packageName    = 'sentinel'
  unzipLocation  = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
  url64bit       = 'https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-windows-amd64.zip'
  checksum64     = '62bad530eabc4bfc3413c478e735d1c40d21ca99b8d9edc6b18af87ee16e507c'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
