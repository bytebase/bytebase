import configs

def test_get_projects(owner_session):
    resp = owner_session.get(f'{configs.BYTEBASE_URL}/api/project')
    assert resp.status_code == 200


def test_create_projects(owner_session):
    creation = {
        'data': {
            "type": 'ProjectCreate',
            "attributes": {
                "name": "My New Project",
                "key": "my-new-project",
            }
        }
    }
    resp = owner_session.post(f'{configs.BYTEBASE_URL}/api/project', json=creation)
    assert resp.status_code == 200