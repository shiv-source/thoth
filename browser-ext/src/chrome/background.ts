import { initBackground } from '../core/background'
import { ext, webextStorage } from '../core/webext'

initBackground({ api: ext, storage: webextStorage })
