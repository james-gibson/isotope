# Burned test key — DO NOT USE

This directory holds a keypair whose **private half is published on purpose**.
Anything signed with it is worthless: that is the point. It exists so that the
trust tests are reproducible — anyone can check the vectors without asking for a
key, and CI exercises the same bytes on every machine.

| file | what it is |
| --- | --- |
| `id_ed25519` | the burned private key (OpenSSH format, no passphrase) |
| `id_ed25519.pub` | its public half, the way `github.com/<user>.keys` serves one |
| `allowed_signers` | the same public key in `ssh-keygen -Y verify` form |
| `note.md` | a one-line "personal note" |
| `note.md.sig` | that note, signed with namespace `lezz-note` |

## Verify it yourself, offline

```sh
ssh-keygen -Y verify -f allowed_signers -I isotope-burned -n lezz-note \
    -s note.md.sig < note.md
# Good "lezz-note" signature for isotope-burned with ED25519 key SHA256:w3+sxEvESgsmFobmbBlpM0yuslXz0pgS+2TLZOjJDgw
```

## Sign something new with it

```sh
cp id_ed25519 /tmp/burned && chmod 600 /tmp/burned      # ssh-keygen wants 0600
ssh-keygen -Y sign -n lezz-note -f /tmp/burned notes.md
```

## Rules

- Never use this key outside `go test` and the vectors above.
- Never add it to a GitHub account, a cluster, or a trust set that matters.
- A test that needs a signature that *should* be refused uses a different,
  freshly generated key — not this one.
- The fingerprint is quoted in the tests, so rotating this key is a deliberate
  change with a diff, not a silent regeneration.
