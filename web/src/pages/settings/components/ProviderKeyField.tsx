import { Flex, Form, Input, Tag } from 'antd'
import { selectSettings } from '../../../store'
import { useAppSelector } from '../../../store/hooks'

// ProviderKeyField is a provider's API key input; like the shared key it is
// write-only, so an empty value leaves the stored key untouched.
export function ProviderKeyField({ provider }: { provider: string }) {
    const settings = useAppSelector(selectSettings)
    const hasKey = settings.data?.providers?.[provider]?.has_api_key === true

    return (
        <Form.Item
            label={
                <Flex align="center" gap={6}>
                    API key
                    {hasKey ? <Tag color="success">Configured</Tag> : <Tag>Not set</Tag>}
                </Flex>
            }
            name={['providers', provider, 'api_key']}
            extra={hasKey ? 'A key is saved — leave blank to keep it.' : 'Falls back to the shared API key when blank.'}
        >
            <Input.Password placeholder={hasKey ? '•••••••• (saved)' : 'sk-…'} autoComplete="off" />
        </Form.Item>
    )
}
