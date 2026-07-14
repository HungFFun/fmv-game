// Admin API client — bảo vệ bằng X-Admin-Token (DEV stub: 'dev-admin').
// Round-trip nội dung qua StoryFile: getContent → sửa → putContent (seed.Replace validate).
import { ApiError } from './game';

const TOKEN_KEY = 'fmv_admin_token';

export const adminToken = {
  get: () => localStorage.getItem(TOKEN_KEY) || '',
  set: (t) => localStorage.setItem(TOKEN_KEY, t),
};

// Tiện ích DEV: seed token từ ?token=... (chia sẻ link admin nhanh). Chạy 1 lần khi import.
try {
  const q = new URLSearchParams(window.location.search).get('token');
  if (q) adminToken.set(q);
} catch { /* noop */ }

async function req(path, init) {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken.get(), ...(init?.headers) },
  });
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const e = body?.error;
    throw new ApiError(res.status, e?.code ?? 'UNKNOWN', e?.message ?? res.statusText, e?.data);
  }
  return body;
}

export const admin = {
  models: () => req('/api/admin/models'),
  chapters: (modelId) => req(`/api/admin/models/${modelId}/chapters`),
  content: (modelId) => req(`/api/admin/models/${modelId}/content`),
  saveContent: (modelId, storyFile) =>
    req(`/api/admin/models/${modelId}/content`, { method: 'PUT', body: JSON.stringify(storyFile) }),
  saveChapterMap: (chapterId, map) =>
    req(`/api/admin/chapters/${chapterId}/map`, { method: 'PUT', body: JSON.stringify(map) }),
};
