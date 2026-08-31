# Local candidate composition

The local candidate plan selects immutable release documents. Each Contract, Plugin, and Sidecar
entry contains `id`, `version`, and `releaseSha256`. The composer verifies the selected
`release.json`, every declared file, the target artifact, Plugin contract implementation, and the
complete Plugin runtime Sidecar set.

The composer does not read component source repositories, package registry storage, build output,
or checkout state. Source repository and commit fields in the output come from the verified release
documents.

Run the owner command with three absolute paths:

```sh
make compose-local-candidate-plan \
  PLAN=/absolute/local-release-selection.json \
  STORE=/absolute/local/releases \
  OUT=/absolute/local/terminal-vision-candidate-plan.json
```

The output must be in the release-store owner directory so its artifact references remain inside
that directory. An equal second invocation returns `unchanged`. Different bytes at the same output
path fail.
