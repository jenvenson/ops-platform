import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"

export default defineConfig({
  plugins: [react()],
  build: {
    chunkSizeWarningLimit: 650,
  },
  server: {
    host: true,
    port: 5173,
    allowedHosts: ["ops-frontend", "docker-host", "ops-frontend-dev"],
    proxy: {
      "/api": {
        target: "http://ops-backend-dev:8080",
        changeOrigin: true,
      },
      "/auth": {
        target: "http://ops-backend-dev:8080",
        changeOrigin: true,
      },
      "/grafana-proxy": {
        target: "http://10.99.99.172:3000",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/grafana-proxy/, ''),
        headers: {
          'Authorization': 'Basic YWRtaW46WkJuc2VjQDIwMjY='
        }
      }
    }
  },
  // 依赖预构建配置
  optimizeDeps: {
    include: ['react', 'react-dom', 'react-is', 'antd', '@ant-design/icons', 'react-router-dom']
  }
})
