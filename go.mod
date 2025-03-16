module github.com/afbjorklund/go-squashnfs

go 1.24.0

require (
	github.com/CalebQ42/squashfs v1.0.6
	github.com/go-git/go-billy/v5 v5.6.2
	github.com/spf13/cobra v1.9.1
	github.com/willscott/go-nfs v0.0.3
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/therootcompany/xz v1.0.1 // indirect
	github.com/willscott/go-nfs-client v0.0.0-20240104095149-b44639837b00 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

replace (
	github.com/CalebQ42/squashfs => github.com/afbjorklund/go-squashfs v1.0.7-0.20250316171921-a22e00c2230e
	github.com/willscott/go-nfs => github.com/afbjorklund/go-nfs v0.0.0-20250314165931-8cb54901dc4b
)
