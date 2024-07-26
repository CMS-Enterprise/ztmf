/**
 * Component that renders all routes in the application.
 * @module router/router
 * @see {@link dashboard/main} for usage.
 */
import { createHashRouter } from 'react-router-dom'
import ErrorBoundary from '@/components/ErrorBoundary'
import authLoader from './authLoader'
import { RouteIds, Routes } from '@/router/constants'
import HomePageContainer from '@/views/Home/Home'
import Title from '@/views/Title/Title'
import PillarPage from '@/views/PillarTable/PillarTable'
import IdentityPage from '@/views/IdentityPage/IdentityPage'
import DevicesPage from '@/views/DevicesPage/DevicesPage'
import NetworksPage from '@/views/NetworksPage/NetworksPage'
import ApplicationPage from '@/views/ApplicationPage/ApplicationPage'
import DataPage from '@/views/DataPage/DataPage'
/**
 * The hash router for the application that defines routes
 *  and specifies the loaders for routes with dynamic data.
 * @type {React.ComponentType} router - The browser router
 * @see {@link https://reactrouter.com/web/api/BrowserRouter BrowserRouter}
 * @see {@link https://reactrouter.com/en/main/route/loader loader}
 */
const router = createHashRouter([
  {
    // index: true,
    id: RouteIds.ROOT,
    path: Routes.ROOT,
    element: <Title />,
    loader: authLoader,
    children: [
      {
        index: true,
        id: RouteIds.HOME,
        element: <HomePageContainer />,
        errorElement: <ErrorBoundary />,
      },
      {
        path: Routes.PILLARS,
        id: RouteIds.PILLARS,
        element: <PillarPage />,
      },
      {
        id: RouteIds.IDENTITY,
        path: Routes.IDENTITY,
        element: <IdentityPage />,
        errorElement: <ErrorBoundary />,
      },
      {
        id: RouteIds.DEVICES,
        path: Routes.DEVICES,
        element: <DevicesPage />,
        errorElement: <ErrorBoundary />,
      },
      {
        id: RouteIds.NETWORKS,
        path: Routes.NETWORKS,
        element: <NetworksPage />,
        errorElement: <ErrorBoundary />,
      },
      {
        id: RouteIds.APPLICATIONS,
        path: Routes.APPLICATIONS,
        element: <ApplicationPage />,
        errorElement: <ErrorBoundary />,
      },
      {
        id: RouteIds.DATA,
        path: Routes.DATA,
        element: <DataPage />,
        errorElement: <ErrorBoundary />,
      },
    ],
  },
])

export default router
