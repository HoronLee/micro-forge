# AGENTS.md - web/ Vue 3 前端应用

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-02-26 -->

## 目录概述

`web/` 是 servora 项目的前端应用，采用现代化的 Vue 3 技术栈。这是一个独立的前端项目，通过 HTTP API 与后端服务通信，支持开发热重载、单元测试和端到端测试。

**核心价值**：
- 现代化技术栈：Vue 3 Composition API + TypeScript + Vite
- 类型安全：完整的 TypeScript 类型系统，禁止 `any` 和 `@ts-ignore`
- 测试完备：Vitest 单元测试 + Playwright E2E 测试
- 开发体验：Vite 快速热重载，ESLint + Prettier 代码规范

## 技术栈

**核心框架**：
- **Vue 3** - 使用 Composition API（`<script setup>`）
- **Vite** - 下一代前端构建工具（快速热重载）
- **TypeScript** - 类型安全的 JavaScript 超集

**状态管理与路由**：
- **Pinia** - Vue 3 官方推荐的状态管理库
- **Vue Router** - Vue 官方路由管理器

**测试框架**：
- **Vitest** - 基于 Vite 的单元测试框架（与 Jest 兼容）
- **Playwright** - 跨浏览器端到端测试框架

**代码质量**：
- **ESLint** - JavaScript/TypeScript 代码检查
- **Prettier** - 代码格式化工具
- **TypeScript Compiler** - 类型检查

## 目录结构

```
web/
├── src/                        # 前端源码
│   ├── components/            # Vue 组件（可复用组件）
│   ├── views/                 # 页面组件（路由对应的页面）
│   ├── router/                # Vue Router 路由配置
│   │   └── index.ts          # 路由定义
│   ├── stores/                # Pinia 状态管理
│   ├── api/                   # API 客户端封装
│   ├── utils/                 # 工具函数
│   ├── assets/                # 静态资源（图片、样式）
│   ├── __tests__/             # Vitest 单元测试
│   ├── App.vue                # 根组件
│   ├── main.ts                # 应用入口
│   ├── style.css              # 全局样式
│   ├── auto-imports.d.ts      # 自动导入类型定义（自动生成）
│   └── components.d.ts        # 组件类型定义（自动生成）
├── e2e/                       # Playwright E2E 测试
│   └── example.spec.ts        # E2E 测试示例
├── public/                    # 公共静态资源（不经过构建）
├── vite.config.ts             # Vite 配置
├── playwright.config.ts       # Playwright 配置
├── eslint.config.ts           # ESLint 配置
├── tsconfig.json              # TypeScript 配置（主配置）
├── tsconfig.app.json          # TypeScript 应用配置
├── tsconfig.node.json         # TypeScript Node 配置
├── tsconfig.vitest.json       # TypeScript Vitest 配置
├── vitest.config.ts           # Vitest 配置
├── package.json               # 依赖配置
├── pnpm-lock.yaml             # pnpm 锁文件
├── env.d.ts                   # 环境变量类型定义
└── README.md                  # 前端文档
```

## 常用命令

### 开发

```bash
cd /Users/horonlee/projects/go/servora/web

# 安装依赖
bun install
# 或
pnpm install

# 启动开发服务器（热重载）
bun dev
# 默认运行在 http://localhost:5173

# 构建生产版本
bun build
# 输出到 dist/ 目录

# 预览生产构建
bun preview
```

### 测试

```bash
# 单元测试（Vitest）
bun test:unit
bun test:unit --watch             # 监听模式
bun test:unit --coverage          # 生成覆盖率报告
bun test:unit src/__tests__/component.spec.ts  # 运行单个测试文件

# E2E 测试（Playwright）
npx playwright install            # 首次安装浏览器（只需运行一次）
bun test:e2e                      # 运行所有 E2E 测试
bun test:e2e e2e/login.spec.ts    # 运行单个测试文件
bun test:e2e --project=chromium   # 只在 Chromium 上运行
bun test:e2e --ui                 # 使用 Playwright UI 模式
bun test:e2e --debug              # 调试模式
```

### 代码质量

```bash
# 代码检查（ESLint）
bun lint
bun lint --fix                    # 自动修复可修复的问题

# 代码格式化（Prettier，通常由 ESLint 调用）
bun format

# 类型检查（TypeScript）
bun type-check
# 或
tsc --noEmit
```

## TypeScript 规范

### 组件示例

使用 `<script setup lang="ts">` Composition API：

```vue
<!-- src/components/UserProfile.vue -->
<script setup lang="ts">
import { ref, computed } from 'vue'

// 定义接口
interface User {
  id: number
  username: string
  email: string
}

// Props 类型
interface Props {
  userId: number
}

const props = defineProps<Props>()

// Emits 类型
interface Emits {
  (e: 'update', user: User): void
  (e: 'delete', id: number): void
}

const emit = defineEmits<Emits>()

// 状态（必须类型化）
const user = ref<User | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

// 计算属性
const displayName = computed(() => {
  return user.value?.username || 'Guest'
})

// 方法（必须类型化返回值）
async function fetchUser(): Promise<void> {
  loading.value = true
  error.value = null

  try {
    const response = await fetch(`/api/users/${props.userId}`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    user.value = await response.json() as User
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
    console.error('Failed to fetch user:', err)
  } finally {
    loading.value = false
  }
}

// 生命周期钩子
onMounted(() => {
  fetchUser()
})

// 发射事件
function handleUpdate() {
  if (user.value) {
    emit('update', user.value)
  }
}
</script>

<template>
  <div class="user-profile">
    <div v-if="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="user">
      <h2>{{ displayName }}</h2>
      <p>Email: {{ user.email }}</p>
      <button @click="handleUpdate">Update</button>
    </div>
  </div>
</template>

<style scoped>
.user-profile {
  padding: 1rem;
}

.error {
  color: red;
}
</style>
```

### 禁止使用的模式

```typescript
// ❌ 禁止使用 as any
const data = response as any
const value = obj.someProperty as any

// ❌ 禁止使用 @ts-ignore
// @ts-ignore
const value = obj.unknownProperty

// ❌ 禁止使用 any 类型
function process(data: any) { }
const items: any[] = []

// ✅ 正确：使用明确的类型
const data = response as User
const value = obj as Record<string, unknown>

// ✅ 正确：使用类型守卫
function isUser(obj: unknown): obj is User {
  return typeof obj === 'object' && obj !== null && 'id' in obj
}

if (isUser(data)) {
  console.log(data.username)
}

// ✅ 正确：使用泛型
function process<T>(data: T): T {
  return data
}

const items: User[] = []
```

### Pinia Store 示例

```typescript
// src/stores/user.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface User {
  id: number
  username: string
  email: string
  token?: string
}

export const useUserStore = defineStore('user', () => {
  // State
  const currentUser = ref<User | null>(null)
  const isAuthenticated = ref(false)

  // Getters（使用 computed）
  const username = computed(() => currentUser.value?.username ?? 'Guest')
  const userId = computed(() => currentUser.value?.id ?? 0)

  // Actions
  async function login(username: string, password: string): Promise<void> {
    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })

      if (!response.ok) {
        throw new Error('Login failed')
      }

      const data = await response.json() as { token: string; user: User }
      currentUser.value = data.user
      isAuthenticated.value = true

      // 保存 token
      localStorage.setItem('auth_token', data.token)
    } catch (error) {
      console.error('Login error:', error)
      throw error
    }
  }

  function logout(): void {
    currentUser.value = null
    isAuthenticated.value = false
    localStorage.removeItem('auth_token')
  }

  return {
    // State
    currentUser,
    isAuthenticated,
    // Getters
    username,
    userId,
    // Actions
    login,
    logout,
  }
})
```

### API 客户端示例

```typescript
// src/api/auth.ts
interface LoginRequest {
  username: string
  password: string
}

interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
    email: string
  }
}

interface RegisterRequest {
  username: string
  password: string
  email: string
}

export const authApi = {
  async login(req: LoginRequest): Promise<LoginResponse> {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    })

    if (!response.ok) {
      const error = await response.json() as { message: string }
      throw new Error(error.message || `Login failed: ${response.statusText}`)
    }

    return response.json() as Promise<LoginResponse>
  },

  async register(req: RegisterRequest): Promise<void> {
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    })

    if (!response.ok) {
      const error = await response.json() as { message: string }
      throw new Error(error.message || `Registration failed: ${response.statusText}`)
    }
  },

  async logout(): Promise<void> {
    const response = await fetch('/api/auth/logout', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('auth_token')}`,
      },
    })

    if (!response.ok) {
      throw new Error(`Logout failed: ${response.statusText}`)
    }
  },
}
```

### Vue Router 示例

```typescript
// src/router/index.ts
import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('../views/DashboardView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('../views/NotFoundView.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const isAuthenticated = localStorage.getItem('auth_token') !== null

  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})

export default router
```

## 测试

### Vitest 单元测试

```typescript
// src/__tests__/components/UserProfile.spec.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import UserProfile from '@/components/UserProfile.vue'

describe('UserProfile.vue', () => {
  beforeEach(() => {
    // 清理 mock
    vi.clearAllMocks()
  })

  it('renders user information when loaded', async () => {
    const mockUser = {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
    }

    // Mock fetch
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockUser),
      } as Response)
    )

    const wrapper = mount(UserProfile, {
      props: { userId: 1 },
    })

    // 等待异步操作完成
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(wrapper.text()).toContain('testuser')
    expect(wrapper.text()).toContain('test@example.com')
  })

  it('displays error message when fetch fails', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: false,
        status: 404,
      } as Response)
    )

    const wrapper = mount(UserProfile, {
      props: { userId: 999 },
    })

    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(wrapper.text()).toContain('error')
  })

  it('emits update event when button clicked', async () => {
    const mockUser = {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockUser),
      } as Response)
    )

    const wrapper = mount(UserProfile, {
      props: { userId: 1 },
    })

    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    await wrapper.find('button').trigger('click')

    expect(wrapper.emitted('update')).toBeTruthy()
    expect(wrapper.emitted('update')?.[0]).toEqual([mockUser])
  })
})
```

### Playwright E2E 测试

```typescript
// e2e/login.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Login Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/login')
  })

  test('should display login form', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Login')
    await expect(page.locator('input[name="username"]')).toBeVisible()
    await expect(page.locator('input[name="password"]')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toBeVisible()
  })

  test('should show error for invalid credentials', async ({ page }) => {
    await page.fill('input[name="username"]', 'wronguser')
    await page.fill('input[name="password"]', 'wrongpass')
    await page.click('button[type="submit"]')

    await expect(page.locator('.error-message')).toContainText('Invalid credentials')
  })

  test('should login successfully with valid credentials', async ({ page }) => {
    await page.fill('input[name="username"]', 'admin')
    await page.fill('input[name="password"]', 'password123')
    await page.click('button[type="submit"]')

    // 等待导航到 dashboard
    await page.waitForURL('**/dashboard')
    await expect(page.locator('h1')).toContainText('Dashboard')
  })

  test('should persist login state after refresh', async ({ page, context }) => {
    await page.fill('input[name="username"]', 'admin')
    await page.fill('input[name="password"]', 'password123')
    await page.click('button[type="submit"]')

    await page.waitForURL('**/dashboard')

    // 刷新页面
    await page.reload()

    // 应该仍然在 dashboard 页面
    await expect(page).toHaveURL(/.*dashboard/)
    await expect(page.locator('h1')).toContainText('Dashboard')
  })
})
```

## AI Agent 工作指南

### 添加新页面

**场景**：添加一个新的产品列表页面

**步骤**：

1. **创建页面组件**
```vue
<!-- src/views/ProductListView.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Product {
  id: number
  name: string
  price: number
}

const products = ref<Product[]>([])
const loading = ref(false)

async function fetchProducts(): Promise<void> {
  loading.value = true
  try {
    const response = await fetch('/api/products')
    products.value = await response.json() as Product[]
  } catch (error) {
    console.error('Failed to fetch products:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProducts()
})
</script>

<template>
  <div class="product-list">
    <h1>Products</h1>
    <div v-if="loading">Loading...</div>
    <ul v-else>
      <li v-for="product in products" :key="product.id">
        {{ product.name }} - ${{ product.price }}
      </li>
    </ul>
  </div>
</template>
```

2. **添加路由**
```typescript
// src/router/index.ts
const routes: RouteRecordRaw[] = [
  // ... 现有路由
  {
    path: '/products',
    name: 'products',
    component: () => import('../views/ProductListView.vue'),
  },
]
```

3. **创建 API 客户端**
```typescript
// src/api/product.ts
interface Product {
  id: number
  name: string
  price: number
}

export const productApi = {
  async getAll(): Promise<Product[]> {
    const response = await fetch('/api/products')
    if (!response.ok) {
      throw new Error('Failed to fetch products')
    }
    return response.json() as Promise<Product[]>
  },

  async getById(id: number): Promise<Product> {
    const response = await fetch(`/api/products/${id}`)
    if (!response.ok) {
      throw new Error('Failed to fetch product')
    }
    return response.json() as Promise<Product>
  },
}
```

4. **编写测试**
```typescript
// e2e/products.spec.ts
import { test, expect } from '@playwright/test'

test('should display product list', async ({ page }) => {
  await page.goto('http://localhost:5173/products')

  await expect(page.locator('h1')).toContainText('Products')
  await expect(page.locator('ul li')).toHaveCount(3)
})
```

### 添加 Pinia Store

**场景**：添加购物车状态管理

**步骤**：

1. **创建 Store**
```typescript
// src/stores/cart.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface CartItem {
  productId: number
  name: string
  price: number
  quantity: number
}

export const useCartStore = defineStore('cart', () => {
  const items = ref<CartItem[]>([])

  const totalItems = computed(() => {
    return items.value.reduce((sum, item) => sum + item.quantity, 0)
  })

  const totalPrice = computed(() => {
    return items.value.reduce((sum, item) => sum + item.price * item.quantity, 0)
  })

  function addItem(item: Omit<CartItem, 'quantity'>): void {
    const existing = items.value.find(i => i.productId === item.productId)
    if (existing) {
      existing.quantity++
    } else {
      items.value.push({ ...item, quantity: 1 })
    }
  }

  function removeItem(productId: number): void {
    const index = items.value.findIndex(i => i.productId === productId)
    if (index > -1) {
      items.value.splice(index, 1)
    }
  }

  function clear(): void {
    items.value = []
  }

  return {
    items,
    totalItems,
    totalPrice,
    addItem,
    removeItem,
    clear,
  }
})
```

2. **在组件中使用**
```vue
<script setup lang="ts">
import { useCartStore } from '@/stores/cart'

const cartStore = useCartStore()

function handleAddToCart() {
  cartStore.addItem({
    productId: 1,
    name: 'Product A',
    price: 99.99,
  })
}
</script>

<template>
  <div>
    <button @click="handleAddToCart">Add to Cart</button>
    <p>Total Items: {{ cartStore.totalItems }}</p>
    <p>Total Price: ${{ cartStore.totalPrice.toFixed(2) }}</p>
  </div>
</template>
```

### 配置代理（解决 CORS）

**场景**：开发环境中访问后端 API

**步骤**：

1. **配置 Vite 代理**
```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8000',  // 后端服务地址
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
```

2. **使用 API**
```typescript
// 开发环境：请求 /api/users → 转发到 http://localhost:8000/users
const response = await fetch('/api/users')
```

### 环境变量配置

**场景**：配置不同环境的 API 地址

**步骤**：

1. **创建环境变量文件**
```bash
# .env.development
VITE_API_BASE_URL=http://localhost:8000

# .env.production
VITE_API_BASE_URL=https://api.example.com
```

2. **使用环境变量**
```typescript
// src/config.ts
export const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL as string,
}

// src/api/client.ts
import { config } from '@/config'

export async function apiRequest(path: string, options?: RequestInit) {
  const url = `${config.apiBaseUrl}${path}`
  return fetch(url, options)
}
```

3. **类型定义**
```typescript
// env.d.ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
```

## 常见任务速查

### 开发工作流

```bash
# 1. 启动开发服务器
cd /Users/horonlee/projects/go/servora/web
bun dev

# 2. 新建组件（手动创建）
mkdir -p src/components
touch src/components/MyComponent.vue

# 3. 新建页面和路由
mkdir -p src/views
touch src/views/MyView.vue
# 编辑 src/router/index.ts 添加路由

# 4. 运行测试
bun test:unit --watch
bun test:e2e

# 5. 检查代码质量
bun lint
bun type-check
```

### 构建和部署

```bash
# 构建生产版本
bun build

# 预览生产构建
bun preview

# 部署到静态托管（如 Nginx）
# 将 dist/ 目录的内容复制到服务器
rsync -avz dist/ user@server:/var/www/html/
```

### 调试技巧

```bash
# 使用 Vue DevTools（浏览器扩展）
# Chrome: https://chrome.google.com/webstore/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd

# Playwright 调试模式
bun test:e2e --debug

# Playwright UI 模式
bun test:e2e --ui

# 查看 Vite 构建分析
bun build --mode analyze
```

## 注意事项

### TypeScript 规范
- 所有组件必须使用 `<script setup lang="ts">`
- 禁止使用 `as any` 或 `@ts-ignore`
- 所有 API 响应必须定义接口类型
- Props 和 Emits 必须明确类型化

### 性能优化
- 使用路由懒加载（`() => import('./View.vue')`）
- 大型列表使用虚拟滚动
- 图片使用懒加载
- 使用 `v-memo` 优化渲染

### 测试覆盖
- 所有 API 调用应有单元测试
- 关键用户流程应有 E2E 测试
- 组件应有快照测试（可选）

### 代码风格
- 遵循 ESLint 规则
- 使用 Prettier 格式化
- 组件命名使用 PascalCase
- 文件命名使用 PascalCase（组件）或 camelCase（工具）

## 依赖关系

**上游依赖**（本目录依赖的其他目录）：
- `/Users/horonlee/projects/go/servora/app/servora/service/` - 后端 API 服务（通过 HTTP 调用）

**外部依赖**：
- Vue 3 框架
- Vite 构建工具
- TypeScript 编译器
- Pinia 状态管理
- Vue Router 路由管理
- Vitest 测试框架
- Playwright E2E 测试

**开发依赖**：
- ESLint（代码检查）
- Prettier（代码格式化）
- Vue DevTools（调试工具）

## 快速参考

**创建新组件**：
```bash
# 创建组件文件
touch src/components/MyButton.vue

# 编写组件
# <script setup lang="ts">
# interface Props {
#   label: string
# }
# defineProps<Props>()
# </script>
```

**创建新页面**：
```bash
# 创建页面文件
touch src/views/AboutView.vue

# 添加路由（src/router/index.ts）
# {
#   path: '/about',
#   name: 'about',
#   component: () => import('../views/AboutView.vue'),
# }
```

**运行开发服务器**：
```bash
cd /Users/horonlee/projects/go/servora/web
bun install  # 首次安装依赖
bun dev      # 启动开发服务器
```
