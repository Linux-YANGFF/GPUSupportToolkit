import { expect, test } from '@playwright/test'

test('GST log analysis page renders', async ({ page }) => {
  await page.goto('/logs.html')

  await expect(page).toHaveTitle(/日志分析 - GST GPU Support Toolkit/)
  await expect(page.getByRole('heading', { name: '日志分析' })).toBeVisible()
  await expect(page.getByPlaceholder('输入日志文件路径...')).toBeVisible()
  await expect(page.getByRole('button', { name: '诊断' })).toHaveCount(0)
})

test('frame detail shows raw log text and downloads frame log', async ({ page }) => {
  await page.goto('/logs.html')

  await page.getByPlaceholder('输入日志文件路径...').fill('/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_demo_api.txt')
  await page.getByRole('button', { name: /解析/ }).click()

  await expect(page.getByRole('heading', { name: '帧列表' })).toBeVisible()
  await expect(page.locator('tbody tr.clickable-row').first()).toBeVisible({ timeout: 20_000 })
  await page.locator('tbody tr.clickable-row').first().click()

  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByText('原始 API 日志')).toBeVisible()
  await expect(page.locator('.raw-log-text')).toContainText(/gl[A-Za-z0-9]+/)

  const firstRawLine = await page.locator('.raw-log-text').innerText()
  expect(firstRawLine.split('\n')[0]).not.toMatch(/^\[\s*\d+\]/)

  const download = page.waitForEvent('download')
  await page.getByRole('button', { name: /下载完整帧日志/ }).click()
  await download
})
