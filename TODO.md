## core

- [x] respect topological order when starting and stopping processes on request
- [x] allow empty commands
- [x] oneshot attribute
- [ ] file tree watching
- [ ] add `-a` (`--all`) flag to `gopmctl logs` command to print all the available logs for a process
- [x] implement process signaling

## docs
- [ ] tutorial
- [ ] reference

## config
- [ ] find configuration file automatically if none is explicitly specified
- [ ] `gopm init` command to initialize a configuration file
- [ ] support individual config file rather than just directory?
- [ ] provide `gopm` package as overlay when reading configuration so users don't need to explicitly vendor it (and risk it getting out of date)
- [ ] print configuration errors consistently and return them all via gRPC so that a `gopmctl reload` command will see them

# later
- [ ] set user ID when specified
