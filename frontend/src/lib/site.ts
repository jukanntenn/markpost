import { version } from '../../package.json'

// Site-wide facts shared by the landing page and the app shell footer:
// external project links and the release version (bumped in package.json by
// the release flow; the only source — never hardcode it elsewhere).
export const APP_VERSION: string = version

export const REPO_URL = 'https://github.com/jukanntenn/markpost'
export const DOCS_URL = 'https://github.com/jukanntenn/markpost#readme'
export const ISSUES_URL = 'https://github.com/jukanntenn/markpost/issues'
export const LICENSE_URL = `${REPO_URL}/blob/main/LICENSE`
export const DOCKER_HUB_URL = 'https://hub.docker.com/r/jukanntenn/markpost'
