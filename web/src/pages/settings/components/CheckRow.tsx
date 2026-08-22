import { Flex, theme, Typography } from 'antd'
import { CheckCircleFilled, CloseCircleFilled } from '@ant-design/icons'
import type { DoctorCheck } from '../../../api/client'

// CheckRow renders one doctor check as a bordered Flex row (List is
// deprecated in antd v6) with a themed status icon and name.
export function CheckRow({ name, ok, message }: DoctorCheck) {
    const { token } = theme.useToken()

    return (
        <Flex align="flex-start" gap={10} className="rounded-md border border-line p-3">
            {ok ? (
                <CheckCircleFilled
                    aria-hidden="true"
                    style={{ color: token.colorSuccess, fontSize: 16, marginTop: 2 }}
                />
            ) : (
                <CloseCircleFilled aria-hidden="true" style={{ color: token.colorError, fontSize: 16, marginTop: 2 }} />
            )}
            <Flex vertical flex={1} className="min-w-0" gap={2}>
                <Typography.Text strong>{name}</Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {message}
                </Typography.Text>
            </Flex>
        </Flex>
    )
}
