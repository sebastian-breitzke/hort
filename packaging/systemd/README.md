# Hort Daemon — systemd (Linux)

`hort daemon install` renders this unit template with the running binary's
path, writes it to `~/.config/systemd/user/hort-daemon.service`, and runs
`systemctl --user daemon-reload && systemctl --user enable --now
hort-daemon.service`.

## Install

```bash
hort daemon install              # user unit, default — idempotent
hort daemon install --system     # /etc/systemd/system (requires root)
hort daemon uninstall            # disable + remove unit
```

## Files

- `hort-daemon.service.tmpl` — Go `text/template` source. Embedded into the
  binary at build time and rendered at install time.

## Notes

- User units start on first login. If you want the daemon up on boot before
  any login, enable linger with `loginctl enable-linger $USER`.
- System installs set `WantedBy=multi-user.target` and run as whatever user
  you pair them with — a system-wide hort install is uncommon since the vault
  itself is per-user; prefer the user-level unit.
