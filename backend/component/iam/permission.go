package iam

type Permission string

const (
	PermissionInstanceList     Permission = "bb.instance.list"
	PermissionInstanceGet      Permission = "bb.instance.get"
	PermissionInstanceCreate   Permission = "bb.instance.create"
	PermissionInstanceUpdate   Permission = "bb.instance.update"
	PermissionInstanceDelete   Permission = "bb.instance.delete"
	PermissionInstanceUndelete Permission = "bb.instance.undelete"
	PermissionInstanceSync     Permission = "bb.instance.sync"
)
