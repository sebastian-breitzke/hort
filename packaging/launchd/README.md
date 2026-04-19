# Hort Daemon — launchd (macOS)

`hort daemon install` renders this plist template with the running binary's
path, drops it into `~/Library/LaunchAgents/tech.s16e.hort.daemon.plist`, and
`launchctl bootstrap`s the agent. That's the whole story — you shouldn't need
to touch these files by hand.

## Manual install

Rare. Prefer `hort daemon install`. Only needed when packaging hort for a
distribution channel that can't execute arbitrary install hooks (cask
post-install is fine — it _can_ run `hort daemon install`).

```bash
hort daemon install              # user agent, default — idempotent
hort daemon install --system     # /Library/LaunchDaemons (requires root)
hort daemon uninstall            # bootout + remove plist
```

## Files

- `tech.s16e.hort.daemon.plist.tmpl` — Go `text/template` source. Embedded
  into the binary at build time and rendered at install time with the real
  `os.Executable()` path and user `$HOME`.

## Reload after upgrading hort

```bash
brew upgrade hort
hort daemon install    # re-renders the plist with the new binary path and bootstraps again
```

## Notes

- `KeepAlive=true` + `ThrottleInterval=10` restart the daemon on crash without
  tight loops.
- The daemon holds no decryption state at rest. Session keys live under
  `~/.hort/{,sources/}*.session` (0600) and are loaded lazily per request, so
  `hort lock` takes effect immediately without a daemon restart.
