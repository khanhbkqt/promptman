import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import type { TimelinePoint } from '@/types/stress'

interface StressChartProps {
  data: TimelinePoint[]
}

export function StressChart({ data }: StressChartProps) {
  if (data.length === 0) return null

  return (
    <Card>
      <CardHeader className="py-3 px-4">
        <CardTitle className="text-sm font-medium">Live Metrics</CardTitle>
      </CardHeader>
      <CardContent className="px-2 pb-4">
        <ResponsiveContainer width="100%" height={320}>
          <LineChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="hsl(var(--border))"
              opacity={0.3}
            />
            <XAxis
              dataKey="elapsed"
              tick={{ fontSize: 10, fill: 'hsl(var(--muted-foreground))' }}
              tickFormatter={(v) => `${Math.round(v)}s`}
              stroke="hsl(var(--border))"
            />

            {/* Left Y-axis: RPS */}
            <YAxis
              yAxisId="rps"
              orientation="left"
              tick={{ fontSize: 10, fill: 'hsl(var(--muted-foreground))' }}
              stroke="hsl(var(--border))"
              label={{
                value: 'RPS',
                angle: -90,
                position: 'insideLeft',
                style: { fontSize: 10, fill: 'hsl(var(--muted-foreground))' },
              }}
            />

            {/* Right Y-axis: Latency (ms) */}
            <YAxis
              yAxisId="latency"
              orientation="right"
              tick={{ fontSize: 10, fill: 'hsl(var(--muted-foreground))' }}
              stroke="hsl(var(--border))"
              label={{
                value: 'p95 (ms)',
                angle: 90,
                position: 'insideRight',
                style: { fontSize: 10, fill: 'hsl(var(--muted-foreground))' },
              }}
            />

            <Tooltip
              contentStyle={{
                backgroundColor: 'hsl(var(--card))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '6px',
                fontSize: '11px',
              }}
              labelFormatter={(v) => `${Math.round(Number(v))}s elapsed`}
            />

            <Legend
              wrapperStyle={{ fontSize: '11px', paddingTop: '8px' }}
            />

            <Line
              yAxisId="rps"
              type="monotone"
              dataKey="rps"
              stroke="#3b82f6"
              strokeWidth={2}
              dot={false}
              name="RPS"
              animationDuration={200}
            />
            <Line
              yAxisId="latency"
              type="monotone"
              dataKey="p95"
              stroke="#f59e0b"
              strokeWidth={2}
              dot={false}
              name="p95 Latency"
              animationDuration={200}
            />
            <Line
              yAxisId="rps"
              type="monotone"
              dataKey="errorRate"
              stroke="#ef4444"
              strokeWidth={1.5}
              dot={false}
              name="Error Rate %"
              animationDuration={200}
            />
            <Line
              yAxisId="rps"
              type="monotone"
              dataKey="activeUsers"
              stroke="#8b5cf6"
              strokeWidth={1.5}
              strokeDasharray="4 2"
              dot={false}
              name="Active Users"
              animationDuration={200}
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
