package api

type projectSettingType string

type LGTMValue string

const (
	projectSettingTypeLGTM projectSettingType = "bb.project.setting.lgtm"
)

type ProjectSetting struct {
	ID int `jsonapi:"primary,policy"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID int
	Project   *Project `jsonapi:"relation,environment"`

	// Domain specific fields
	Type    projectSettingType `jsonapi:"attr,type"`
	Payload string             `jsonapi:"attr,payload"`
}

type ProjectSettingFind struct {
	ProjectID *int
	Type      *projectSettingType
}

type ProjectSettingUpsert struct {
}
