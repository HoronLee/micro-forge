<template>
  <div class="min-h-screen bg-gray-50 py-12 px-4">
    <div class="max-w-md mx-auto">
      <h1 class="text-3xl font-bold text-center mb-8">欢迎来到 servora</h1>

      <div
        v-if="!authStore.isAuthenticated()"
        class="bg-white rounded-lg shadow p-6"
      >
        <el-tabs v-model="activeTab">
          <el-tab-pane label="登录" name="login">
            <el-form @submit.prevent="handleLogin">
              <el-form-item label="邮箱">
                <el-input v-model="email" placeholder="请输入邮箱" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input
                  v-model="password"
                  type="password"
                  placeholder="请输入密码"
                />
              </el-form-item>
              <el-button
                type="primary"
                @click="handleLogin"
                :loading="loading"
                class="w-full"
              >
                登录
              </el-button>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="注册" name="signup">
            <el-form @submit.prevent="handleSignup">
              <el-form-item label="用户名">
                <el-input v-model="signupForm.name" placeholder="至少5个字符" />
              </el-form-item>
              <el-form-item label="邮箱">
                <el-input v-model="signupForm.email" placeholder="请输入邮箱" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input
                  v-model="signupForm.password"
                  type="password"
                  placeholder="5-10个字符"
                />
              </el-form-item>
              <el-form-item label="确认密码">
                <el-input
                  v-model="signupForm.passwordConfirm"
                  type="password"
                  placeholder="再次输入密码"
                />
              </el-form-item>
              <el-button
                type="primary"
                @click="handleSignup"
                :loading="loading"
                class="w-full"
              >
                注册
              </el-button>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </div>

      <div v-else class="bg-white rounded-lg shadow p-6">
        <h2 class="text-xl font-semibold mb-4">用户信息</h2>
        <div class="space-y-2">
          <p><span class="font-medium">ID:</span> {{ authStore.user?.id }}</p>
          <p>
            <span class="font-medium">用户名:</span> {{ authStore.user?.name }}
          </p>
          <p>
            <span class="font-medium">角色:</span> {{ authStore.user?.role }}
          </p>
        </div>
        <el-button @click="handleLogout" class="w-full mt-4"
          >退出登录</el-button
        >
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/utils/api";

const authStore = useAuthStore();
const activeTab = ref("login");
const email = ref("");
const password = ref("");
const loading = ref(false);
const signupForm = reactive({
  name: "",
  email: "",
  password: "",
  passwordConfirm: "",
});

const handleLogin = async () => {
  if (!email.value || !password.value) {
    ElMessage.warning("请输入邮箱和密码");
    return;
  }

  loading.value = true;
  try {
    const loginResult = await api.login(email.value, password.value);
    const token = loginResult.accessToken;

    if (!token) {
      throw new Error("登录成功但未返回 access token");
    }

    authStore.setToken(token);
    const userInfo = await api.getCurrentUser(token);
    authStore.setUser({
      id: String(userInfo.id ?? ""),
      name: userInfo.name ?? "",
      role: userInfo.role ?? "",
    });
    ElMessage.success("登录成功");
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : "登录失败");
  } finally {
    loading.value = false;
  }
};

const handleSignup = async () => {
  if (
    !signupForm.name ||
    !signupForm.email ||
    !signupForm.password ||
    !signupForm.passwordConfirm
  ) {
    ElMessage.warning("请填写完整信息");
    return;
  }

  loading.value = true;
  try {
    await api.signup(
      signupForm.name,
      signupForm.email,
      signupForm.password,
      signupForm.passwordConfirm,
    );
    ElMessage.success("注册成功，请登录");
    activeTab.value = "login";
    email.value = signupForm.email;
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : "注册失败");
  } finally {
    loading.value = false;
  }
};

const handleLogout = () => {
  authStore.logout();
  ElMessage.success("已退出登录");
};

onMounted(async () => {
  if (authStore.token) {
    try {
      const userInfo = await api.getCurrentUser(authStore.token);
      authStore.setUser({
        id: String(userInfo.id ?? ""),
        name: userInfo.name ?? "",
        role: userInfo.role ?? "",
      });
    } catch {
      authStore.logout();
    }
  }
});
</script>
