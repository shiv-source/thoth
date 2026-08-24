import { Card, Progress, Statistic } from 'antd'

// WikiStorage is the storage widget's numbers: wiki size on disk and the
// attachments it holds (mock until a size endpoint lands).
export interface WikiStorage {
    sizeMB: number
    attachments: number
    percent: number
}

// WikiStorageCard is the Overview "Storage" widget: the wiki's footprint at a
// glance — size, attachments, and a soft-cap progress bar.
export function WikiStorageCard({ storage }: { storage: WikiStorage }) {
    return (
        <Card size="small" title="Storage">
            <div className="grid grid-cols-2 gap-4">
                <Statistic
                    title="Wiki size"
                    value={storage.sizeMB}
                    precision={1}
                    suffix="MB"
                    styles={{ content: { fontSize: 22 } }}
                />
                <Statistic title="Attachments" value={storage.attachments} styles={{ content: { fontSize: 22 } }} />
            </div>
            <Progress percent={storage.percent} size="small" className="mt-4" strokeColor="var(--ant-color-primary)" />
            <p className="mt-3 text-xs text-subtle">mock data — wiki size on disk</p>
        </Card>
    )
}
