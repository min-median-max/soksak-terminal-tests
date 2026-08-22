param([Parameter(Mandatory=$true)][string]$Stage)
$ErrorActionPreference = "Stop"
$target = "x86_64-pc-windows-msvc"
$plugins = [ordered]@{
  "soksak-plugin-terminal-alacritty"="v0.0.1"; "soksak-plugin-terminal-ghostty"="v0.0.2";
  "soksak-plugin-terminal-vt100"="v0.0.1"; "soksak-plugin-terminal-wezterm"="v0.0.1"; "soksak-plugin-terminal-xterm"="v0.0.3"
}
$sidecars = [ordered]@{
  "soksak-sidecar-pty"="v0.0.1";
  "soksak-sidecar-terminal-alacritty"="v0.0.5"; "soksak-sidecar-terminal-ghostty"="v0.0.5";
  "soksak-sidecar-terminal-vt100"="v0.0.5"; "soksak-sidecar-terminal-wezterm"="v0.0.5"
}
New-Item -ItemType Directory -Force $Stage | Out-Null
$pluginInput=@{}
foreach($id in $plugins.Keys){
  $dir=Join-Path $Stage "downloads/$id"; $install=Join-Path $Stage "plugins/$id"; New-Item -ItemType Directory -Force $dir,$install|Out-Null
  gh release download $plugins[$id] --repo "soksak-ai/$id" --pattern release.json --pattern "*.tgz" --dir $dir
  $release=Get-Content (Join-Path $dir release.json)-Raw|ConvertFrom-Json; $artifact=$release.artifacts[0]; $archive=Join-Path $dir ([IO.Path]::GetFileName($artifact.url))
  if((Get-FileHash $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $artifact.sha256){throw "$id digest mismatch"}; tar -xzf $archive -C $install
  $pluginInput[$id]=@{path=$install;repository=$release.source.repository;commit=$release.source.commit;artifactSha256=$artifact.sha256}
}
$sidecarInput=@{}
foreach($id in $sidecars.Keys){
  $dir=Join-Path $Stage "downloads/$id"; $install=Join-Path $Stage "sidecars/$id"; New-Item -ItemType Directory -Force $dir,$install|Out-Null
  gh release download $sidecars[$id] --repo "soksak-ai/$id" --pattern release.json --pattern "*$target.tar.gz" --dir $dir
  $release=Get-Content (Join-Path $dir release.json)-Raw|ConvertFrom-Json; $artifact=$release.artifacts|Where-Object target -eq $target; $archive=Join-Path $dir ([IO.Path]::GetFileName($artifact.url))
  if((Get-FileHash $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $artifact.sha256){throw "$id digest mismatch"}; tar -xzf $archive -C $install
  $sidecarInput[$id]=@{path=$install;repository=$release.source.repository;commit=$release.source.commit;artifactSha256=$artifact.sha256;target=$target}
}
@{platform="windows";home=(Join-Path $Stage "composition-home");plugins=$pluginInput;sidecars=$sidecarInput}|ConvertTo-Json -Depth 8|Set-Content -Encoding utf8NoBOM (Join-Path $Stage development-input.json)
