import { useEffect } from 'react'
import { Alert, Button, Card, Divider, Flex, Progress, Typography } from 'antd'
import { MedicineBoxOutlined, ReloadOutlined } from '@ant-design/icons'
import { runDoctor, selectDoctorChecks, selectDoctorError, selectDoctorRunning } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CardTitle } from './components/CardTitle'
import { CheckRow } from './components/CheckRow'
import { SettingsShell } from './SettingsShell'

// SettingsDoctorPage runs the shared installation checks (GET /api/doctor)
// on open and on demand, rendering them in one card with a pass summary.
export function SettingsDoctorPage() {
    const dispatch = useAppDispatch()
    const checks = useAppSelector(selectDoctorChecks)
    const running = useAppSelector(selectDoctorRunning)
    const error = useAppSelector(selectDoctorError)

    useEffect(() => {
        void dispatch(runDoctor())
    }, [dispatch])

    const passed = checks?.filter((c) => c.ok).length ?? 0
    const percent = checks && checks.length > 0 ? Math.round((passed / checks.length) * 100) : 0
    const healthy = checks !== null && passed === checks.length

    return (
        <SettingsShell active="doctor">
            <Card size="small" title={<CardTitle icon={MedicineBoxOutlined}>Checks</CardTitle>}>
                {error && <Alert type="error" showIcon message={error} className="mb-4" />}
                <div className="flex items-center justify-between gap-3">
                    <p className="text-sm text-subtle">
                        Installation health, using the same checks as{' '}
                        <code className="font-mono text-xs">thoth doctor</code>.
                    </p>
                    <Button
                        icon={<ReloadOutlined aria-hidden="true" />}
                        loading={running}
                        onClick={() => void dispatch(runDoctor())}
                    >
                        Run checks
                    </Button>
                </div>
                {checks && (
                    <>
                        <Flex align="center" gap={12} className="mt-5 rounded-md border border-line bg-raised p-4">
                            <Flex vertical className="min-w-0 flex-1" gap={2}>
                                <Typography.Text strong className="text-heading">
                                    {healthy
                                        ? 'All systems go'
                                        : `${checks.length - passed} check${checks.length - passed === 1 ? '' : 's'} need attention`}
                                </Typography.Text>
                                <Typography.Text type="secondary" className="text-sm">
                                    {passed} of {checks.length} checks passed
                                </Typography.Text>
                            </Flex>
                            <Progress
                                type="circle"
                                percent={percent}
                                size={56}
                                strokeColor={healthy ? undefined : 'var(--ant-color-warning)'}
                            />
                        </Flex>
                        <Divider />
                        <div className="grid gap-4 md:grid-cols-2">
                            {checks.map((c) => (
                                <CheckRow key={c.name} {...c} />
                            ))}
                        </div>
                    </>
                )}
            </Card>
        </SettingsShell>
    )
}
