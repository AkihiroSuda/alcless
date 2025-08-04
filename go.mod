// gomodjail:confined
module github.com/AkihiroSuda/alcless

go 1.23.0

require (
	al.essio.dev/pkg/shellescape v1.6.0
	github.com/containerd/containerd/v2 v2.1.4
	github.com/lmittmann/tint v1.1.2
	github.com/sethvargo/go-password v0.3.1
	github.com/spf13/cobra v1.9.1 // gomodjail:unconfined
	golang.org/x/term v0.33.0
	gotest.tools/v3 v3.5.2
)

require (
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/sys v0.34.0 // indirect
)
