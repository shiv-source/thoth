import { theme, Typography } from 'antd'
import { CheckCircleFilled, CloseCircleFilled } from '@ant-design/icons'
import type { DoctorCheck } from '../../../api/client'

// CheckRow renders one doctor check as a bordered row (List is deprecated in
// antd v6) with a themed status icon and name. A plain flex div carries the
// padding — antd's own `.ant-flex { padding: 0 }` reset would cancel a
// Tailwind padding utility on a Flex.
export function CheckRow({ name, ok, message }: DoctorCheck) {
    const { token } = theme.useToken()

    return (
        <div className="flex items-start gap-2.5 rounded-md border border-line p-3">
            {ok ? (
                <CheckCircleFilled
                    aria-hidden="true"
                    style={{ color: token.colorSuccess, fontSize: 16, marginTop: 3 }}
                />
            ) : (
                <CloseCircleFilled aria-hidden="true" style={{ color: token.colorError, fontSize: 16, marginTop: 3 }} />
            )}
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <Typography.Text strong>{name}</Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {message}
                </Typography.Text>
            </div>
        </div>
    )
}
