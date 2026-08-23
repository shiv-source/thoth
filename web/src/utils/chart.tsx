import {
    ArcElement,
    BarController,
    BarElement,
    CategoryScale,
    Chart,
    DoughnutController,
    Filler,
    LinearScale,
    LineController,
    LineElement,
    PointElement,
    Tooltip
} from 'chart.js'

// Registers every Chart.js piece the dashboard charts use — exactly once.
// Chart components import this module for its side effect.
Chart.register(
    BarController,
    BarElement,
    CategoryScale,
    LinearScale,
    Tooltip,
    LineController,
    LineElement,
    PointElement,
    ArcElement,
    DoughnutController,
    Filler
)
