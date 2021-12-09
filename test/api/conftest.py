import requests
import pytest

import configs

@pytest.fixture
def owner_session():
	with requests.Session() as sess:
		resp = sess.post(
			f'{configs.BYTEBASE_URL}/api/auth/login',
			json={
				'data': {
					'type': 'loginInfo',
					'attributes': {
						"email": 'demo@example.com',
						"password": '1024'
					}
				}
			})
		if resp.status_code != 200:
			raise RuntimeError(f'Failed to login: {resp.status_code}')
		yield sess