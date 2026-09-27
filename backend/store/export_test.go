package store

// SetSettingPublishHookForTest installs a hook that runs between a setting
// update's commit and its cache publish — test-only, to deterministically
// expose the publish-ordering window.
func (s *Store) SetSettingPublishHookForTest(hook func()) {
	s.settingPublishHookForTest = hook
}
