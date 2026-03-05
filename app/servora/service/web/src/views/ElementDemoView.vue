<template>
  <div class="element-demo">
    <el-container>
      <el-header>
        <h1>Element Plus 示例页面</h1>
      </el-header>

      <el-main>
        <!-- 基础组件 -->
        <el-card class="mb-4" header="基础组件">
          <div class="demo-section">
            <!-- 按钮 -->
            <div class="mb-4">
              <h3>按钮 (Button)</h3>
              <el-space>
                <el-button type="primary">主要按钮</el-button>
                <el-button type="success">成功按钮</el-button>
                <el-button type="warning">警告按钮</el-button>
                <el-button type="danger">危险按钮</el-button>
                <el-button type="info">信息按钮</el-button>
                <el-button>默认按钮</el-button>
              </el-space>
            </div>

            <!-- 标签 -->
            <div class="mb-4">
              <h3>标签 (Tag)</h3>
              <el-space>
                <el-tag>标签一</el-tag>
                <el-tag type="success">成功</el-tag>
                <el-tag type="warning">警告</el-tag>
                <el-tag type="danger">危险</el-tag>
                <el-tag type="info">信息</el-tag>
              </el-space>
            </div>

            <!-- 图标 -->
            <div class="mb-4">
              <h3>图标 (Icon)</h3>
              <el-space size="large">
                <el-icon :size="24"><Edit /></el-icon>
                <el-icon :size="24" color="#409EFF"><Share /></el-icon>
                <el-icon :size="24" color="#67C23A"><Delete /></el-icon>
                <el-icon :size="24" color="#E6A23C"><Search /></el-icon>
                <el-icon :size="24" color="#F56C6C"><Upload /></el-icon>
              </el-space>
            </div>
          </div>
        </el-card>

        <!-- 表单组件 -->
        <el-card class="mb-4" header="表单组件">
          <el-form :model="form" label-width="80px">
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="输入框">
                  <el-input v-model="form.input" placeholder="请输入内容" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="选择器">
                  <el-select v-model="form.select" placeholder="请选择">
                    <el-option label="选项1" value="option1" />
                    <el-option label="选项2" value="option2" />
                    <el-option label="选项3" value="option3" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="日期选择">
                  <el-date-picker
                    v-model="form.date"
                    type="date"
                    placeholder="选择日期"
                    style="width: 100%"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="开关">
                  <el-switch v-model="form.switch" />
                </el-form-item>
              </el-col>
            </el-row>

            <el-form-item label="单选框">
              <el-radio-group v-model="form.radio">
                <el-radio label="option1">选项1</el-radio>
                <el-radio label="option2">选项2</el-radio>
                <el-radio label="option3">选项3</el-radio>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="复选框">
              <el-checkbox-group v-model="form.checkbox">
                <el-checkbox label="check1">复选框1</el-checkbox>
                <el-checkbox label="check2">复选框2</el-checkbox>
                <el-checkbox label="check3">复选框3</el-checkbox>
              </el-checkbox-group>
            </el-form-item>

            <el-form-item label="评分">
              <el-rate v-model="form.rate" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="submitForm">提交</el-button>
              <el-button @click="resetForm">重置</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 数据展示 -->
        <el-card class="mb-4" header="数据展示">
          <!-- 表格 -->
          <div class="mb-4">
            <h3>表格 (Table)</h3>
            <el-table :data="tableData" style="width: 100%">
              <el-table-column prop="date" label="日期" width="180" />
              <el-table-column prop="name" label="姓名" width="180" />
              <el-table-column prop="address" label="地址" />
              <el-table-column fixed="right" label="操作" width="120">
                <template #default>
                  <el-button link type="primary" size="small">编辑</el-button>
                  <el-button link type="danger" size="small">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <!-- 分页 -->
          <div class="mb-4">
            <h3>分页 (Pagination)</h3>
            <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="400"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="handleSizeChange"
              @current-change="handleCurrentChange"
            />
          </div>

          <!-- 树形控件 -->
          <div class="mb-4">
            <h3>树形控件 (Tree)</h3>
            <el-tree :data="treeData" :props="defaultProps" @node-click="handleNodeClick" />
          </div>
        </el-card>

        <!-- 反馈组件 -->
        <el-card class="mb-4" header="反馈组件">
          <div class="demo-section">
            <!-- 进度条 -->
            <div class="mb-4">
              <h3>进度条 (Progress)</h3>
              <el-progress :percentage="percentage" :color="customColor" />
              <div class="mt-2">
                <el-button @click="decrease">减少 10%</el-button>
                <el-button @click="increase">增加 10%</el-button>
              </div>
            </div>

            <!-- 消息提示按钮 -->
            <div class="mb-4">
              <h3>消息提示 (Message)</h3>
              <el-space>
                <el-button @click="openMessage('success')">成功</el-button>
                <el-button @click="openMessage('warning')">警告</el-button>
                <el-button @click="openMessage('info')">信息</el-button>
                <el-button @click="openMessage('error')">错误</el-button>
              </el-space>
            </div>

            <!-- 通知提示按钮 -->
            <div class="mb-4">
              <h3>通知 (Notification)</h3>
              <el-button type="primary" @click="openNotification">打开通知</el-button>
            </div>
          </div>
        </el-card>

        <!-- 导航 -->
        <el-card class="mb-4" header="导航">
          <!-- 菜单 -->
          <div class="mb-4">
            <h3>菜单 (Menu)</h3>
            <el-menu mode="horizontal" :default-active="activeIndex" @select="handleSelect">
              <el-menu-item index="1">处理中心</el-menu-item>
              <el-sub-menu index="2">
                <template #title>工作台</template>
                <el-menu-item index="2-1">选项1</el-menu-item>
                <el-menu-item index="2-2">选项2</el-menu-item>
                <el-menu-item index="2-3">选项3</el-menu-item>
              </el-sub-menu>
              <el-menu-item index="3">消息中心</el-menu-item>
            </el-menu>
          </div>

          <!-- 面包屑 -->
          <div class="mb-4">
            <h3>面包屑 (Breadcrumb)</h3>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
              <el-breadcrumb-item>活动管理</el-breadcrumb-item>
              <el-breadcrumb-item>活动列表</el-breadcrumb-item>
              <el-breadcrumb-item>活动详情</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </el-card>
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'

// 表单数据
const form = reactive({
  input: '',
  select: '',
  date: '',
  switch: false,
  radio: 'option1',
  checkbox: ['check1'],
  rate: 3,
})

// 表格数据
const tableData = [
  {
    date: '2016-05-02',
    name: '王小虎',
    address: '上海市普陀区金沙江路 1518 弄',
  },
  {
    date: '2016-05-04',
    name: '王小虎',
    address: '上海市普陀区金沙江路 1517 弄',
  },
  {
    date: '2016-05-01',
    name: '王小虎',
    address: '上海市普陀区金沙江路 1519 弄',
  },
  {
    date: '2016-05-03',
    name: '王小虎',
    address: '上海市普陀区金沙江路 1516 弄',
  },
]

// 分页
const currentPage = ref(1)
const pageSize = ref(10)
const handleSizeChange = (val: number) => {
  console.log(`每页 ${val} 条`)
}
const handleCurrentChange = (val: number) => {
  console.log(`当前页: ${val}`)
}

// 树形控件
const treeData = [
  {
    label: '一级 1',
    children: [
      {
        label: '二级 1-1',
        children: [
          {
            label: '三级 1-1-1',
          },
        ],
      },
    ],
  },
  {
    label: '一级 2',
    children: [
      {
        label: '二级 2-1',
        children: [
          {
            label: '三级 2-1-1',
          },
        ],
      },
      {
        label: '二级 2-2',
        children: [
          {
            label: '三级 2-2-1',
          },
        ],
      },
    ],
  },
]

const defaultProps = {
  children: 'children',
  label: 'label',
}

interface TreeNode {
  label: string
  children?: TreeNode[]
}

const handleNodeClick = (data: TreeNode) => {
  console.log(data)
}

// 进度条
const percentage = ref(70)
const customColor = ref('#409eff')

const increase = () => {
  percentage.value += 10
  if (percentage.value > 100) {
    percentage.value = 100
  }
}
const decrease = () => {
  percentage.value -= 10
  if (percentage.value < 0) {
    percentage.value = 0
  }
}

// 消息提示
const openMessage = (type: string) => {
  ElMessage({
    message: `这是一条${type === 'success' ? '成功' : type === 'warning' ? '警告' : type === 'info' ? '信息' : '错误'}消息`,
    type: type as 'success' | 'warning' | 'info' | 'error',
  })
}

// 通知
const openNotification = () => {
  ElNotification({
    title: '标题',
    message: '这是一条不会自动关闭的通知',
    duration: 0,
  })
}

// 菜单
const activeIndex = ref('1')
const handleSelect = (key: string, keyPath: string[]) => {
  console.log(key, keyPath)
}

// 表单操作
const submitForm = () => {
  ElMessage.success('表单提交成功！')
  console.log('表单数据：', form)
}

const resetForm = () => {
  ElMessage.info('表单已重置')
  Object.assign(form, {
    input: '',
    select: '',
    date: '',
    switch: false,
    radio: 'option1',
    checkbox: ['check1'],
    rate: 3,
  })
}
</script>

<style scoped>
.element-demo {
  padding: 20px;
}

.el-header {
  background-color: #f5f7fa;
  color: #333;
  text-align: center;
  line-height: 60px;
  border-bottom: 1px solid #e4e7ed;
}

.mb-4 {
  margin-bottom: 24px;
}

.mt-2 {
  margin-top: 8px;
}

.demo-section {
  padding: 16px;
}

h3 {
  margin-bottom: 16px;
  color: #303133;
  font-size: 16px;
  font-weight: 500;
}

.el-card {
  margin-bottom: 20px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
}
</style>
