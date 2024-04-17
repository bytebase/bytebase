# Installation

- Install buf: https://docs.buf.build/installation
- Run: buf generate
- Format: buf format -w

Note:
You may have command conflict for the `buf`: https://github.com/ohmyzsh/ohmyzsh/issues/11169
If so please remove the brew alias in `~/.oh-my-zsh/plugins/brew/brew.plugin.zsh`

# Clients
- Web: grpcui -plaintext localhost:8081
- CLI: grpcurl