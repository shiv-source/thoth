import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ConfigProvider, App as AntApp } from 'antd'
import { ext, webextStorage } from '../core/webext'
import { PopupApp } from './App'
import { popupTheme } from './theme'

const root = document.getElementById('root')
if (root) {
    createRoot(root).render(
        <StrictMode>
            <ConfigProvider theme={popupTheme}>
                <AntApp>
                    <PopupApp storage={webextStorage} ext={ext} />
                </AntApp>
            </ConfigProvider>
        </StrictMode>
    )
}
