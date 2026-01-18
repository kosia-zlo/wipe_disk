# Simple certificate creation script
$cert = New-SelfSignedCertificate -DnsName "WipeDisk_Dev" -CertStoreLocation "cert:\CurrentUser\My" -KeyUsage "DigitalSignature","KeyEncipherment" -NotAfter (Get-Date).AddYears(5) -KeyExportPolicy "Exportable" -KeyLength 2048 -HashAlgorithm "SHA256"
$securePassword = ConvertTo-SecureString -String "WipeDiskDev2026" -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath "WipeDisk_Dev.pfx" -Password $securePassword -Force
Export-Certificate -Cert $cert -FilePath "WipeDisk_Dev.cer" -Force
Write-Host "Certificate created successfully"
