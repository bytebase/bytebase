/*
 * @Author: SIOOOO gesiyuan01@gmail.com
 * @Date: 2023-07-19 23:38:48
 * @LastEditors: SIOOOO gesiyuan01@gmail.com
 * @LastEditTime: 2023-07-20 18:56:49
 * @FilePath: /bytebase/backend/resources/mysql/resources_darwin.go
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%A
 */
//go:build mysql
// +build mysql

package mysql

import "embed"

// f1943053b12428e4c0e4ed309a636fd0 is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.28-macos11-arm64.tar.gz f1943053b12428e4c0e4ed309a636fd0

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.28-macos11-arm64.tar.gz
var resources embed.FS
s