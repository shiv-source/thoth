import { useEffect } from 'react'
import { Alert, Button, Card, Divider, Flex, Progress, Spin, Typography } from 'antd'
import { CheckCircleOutlined, MedicineBoxOutlined, ReloadOutlined } from '@ant-design/icons'
import { runDoctor, selectDoctorChecks, selectDoctorError, selectDoctorRunning } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CardTitle } from './components/CardTitle'
import { CheckRow } from './components/CheckRow'
import { FormSection } from './components/FormSection'
import { SettingsShell } from './SettingsShell'

// SettingsDoctorPage runs the shared installation checks (GET /api/doctor)
// on open and on demand, rendering them in one card with a pass summary and
// the per-check rows grouped in a panel — the same shape as the other tabs.
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
            <Card
                title={<CardTitle icon={MedicineBoxOutlined}>Checks</CardTitle>}
                extra={
                    <Button
                        icon={<ReloadOutlined aria-hidden="true" />}
                        loading={running}
                        onClick={() => void dispatch(runDoctor())}
                    >
                        Run checks
                    </Button>
                }
            >
                {error && <Alert type="error" showIcon title={error} className="mb-4" />}
                {checks === null ? (
                    <div className="flex justify-center py-10">
                        <Spin />
                    </div>
                ) : (
                    <FormSection
                        icon={CheckCircleOutlined}
                        title="Installation health"
                        description="The same checks as thoth doctor — re-run any time after changing providers or the wiki."
                    >
                        <div className="rounded-lg border border-line bg-raised p-5">
                            <Flex align="center" gap={12}>
                                <Flex vertical className="min-w-0 flex-1" gap={2}>
                                    <Typography.Text strong className="text-base text-heading">
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
                            <Divider className="my-5" />
                            <div className="grid gap-3 md:grid-cols-2">
                                {checks.map((c) => (
                                    <CheckRow key={c.name} {...c} />
                                ))}
                            </div>
                        </div>
                    </FormSection>
                )}
            </Card>
        </SettingsShell>
    )
}
