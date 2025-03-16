
GO = go

prefix ?= /usr/local
# macOS doesn't have /usr/local/sbin in PATH
bindir = $(prefix)/bin
sbindir = $(prefix)/bin

all: squashnfsd

squashnfsd: cmd/squashnfsd/main.go
	$(GO) build -o $@ ./cmd/squashnfsd

.PHONY: clean
clean:
	$(RM) squashnfsd

.PHONY: install
install: squashnfs squashnfsd
	install -d $(DESTDIR)$(bindir)
	install squashnfs $(DESTDIR)$(bindir)
	install -d $(DESTDIR)$(sbindir)
	install squashnfsd $(DESTDIR)$(sbindir)
