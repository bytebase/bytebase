<!--
 * @Author: SIOOOO gesiyuan01@gmail.com
 * @Date: 2023-07-19 23:38:48
 * @LastEditors: SIOOOO gesiyuan01@gmail.com
 * @LastEditTime: 2023-07-20 01:09:08
 * @FilePath: /bytebase/proto/README.md
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
-->
# Installation

- Install buf: https://docs.buf.build/installation
- Run: buf generate

Note:
You may have command conflict for the `buf`: https://github.com/ohmyzsh/ohmyzsh/issues/11169
If so please remove the brew alias in `~/.oh-my-zsh/plugins/brew/brew.plugin.zsh`

# Clients
- Web: grpcui -plaintext localhost:8081
- CLI: grpcurl