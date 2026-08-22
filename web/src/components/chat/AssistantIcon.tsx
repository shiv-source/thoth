import { Avatar, theme } from 'antd'
import { RobotOutlined } from '@ant-design/icons'

// AssistantIcon is the small avatar shown to the left of every assistant
// message, mirroring the app's accent color.
export function AssistantIcon() {
    const { token } = theme.useToken()
    return (
        <Avatar
            aria-hidden="true"
            size={28}
            shape="square"
            icon={<RobotOutlined />}
            className="mt-0.5 shrink-0 shadow-sm"
            style={{
                borderRadius: token.borderRadius,
                backgroundColor: token.colorPrimary,
                color: token.colorTextLightSolid
            }}
        />
    )
}
