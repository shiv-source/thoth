import { Button, Layout } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { navigate } from '../../hooks/useConversationRoute'
import { ChatsList } from './ChatsList'

// Sidebar is the chat view's history column: chats label, New chat
// primary button, and the grouped conversation list (ChatsList). The brand
// wordmark and health footer live in AppSider.
export function Sidebar() {
    return (
        <Layout.Sider width={288} theme="light" className="bg-surface">
            <div className="flex h-full flex-col">
                <div className="shrink-0 px-4 pb-1.5 pt-4">
                    <span className="text-xs font-semibold uppercase tracking-wider text-faint">Chats</span>
                </div>
                <div className="shrink-0 px-4 pb-3">
                    <Button
                        type="primary"
                        block
                        icon={<PlusOutlined aria-hidden="true" />}
                        onClick={() => navigate('/chat')}
                    >
                        New chat
                    </Button>
                </div>
                <div className="min-h-0 flex-1">
                    <ChatsList />
                </div>
            </div>
        </Layout.Sider>
    )
}
