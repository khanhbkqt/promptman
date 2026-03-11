import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { AppLayout } from "@/layouts/AppLayout"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import {
  CollectionsPage,
  EnvironmentsPage,
  TestsPage,
  HistoryPage,
  StressPage,
  NotFoundPage,
} from "@/pages"

function App() {
  return (
    <div className="dark min-h-screen bg-background text-foreground">
      <ErrorBoundary>
        <BrowserRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/" element={<Navigate to="/collections" replace />} />
              <Route path="/collections" element={<CollectionsPage />} />
              <Route path="/environments" element={<EnvironmentsPage />} />
              <Route path="/tests" element={<TestsPage />} />
              <Route path="/history" element={<HistoryPage />} />
              <Route path="/stress" element={<StressPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ErrorBoundary>
    </div>
  )
}

export default App
