// Model layer — server là nguồn chân lý, client chỉ giữ cache phiên.
// (Thin-client: dùng typed fetch wrapper, KHÔNG axios/SWR theo ràng buộc frontend-player.)

export class ApiError extends Error {
  constructor(status, code, message, data) {
    super(message);
    this.status = status;
    this.code = code;
    this.data = data;
  }
}

async function request(path, init) {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const e = body?.error;
    throw new ApiError(res.status, e?.code ?? 'UNKNOWN', e?.message ?? res.statusText, e?.data);
  }
  return body;
}

export const api = {
  current: () => request('/api/play/current'),
  jumpScene: (code) => request(`/api/play/scene/${encodeURIComponent(code)}`),
  advance: (choiceId) =>
    request('/api/play/advance', {
      method: 'POST',
      body: JSON.stringify(choiceId != null ? { choiceId } : {}),
    }),
  restart: () => request('/api/play/restart', { method: 'POST', body: '{}' }),
  saves: () => request('/api/saves'),
  saveSlot: (slot) =>
    request('/api/saves', { method: 'POST', body: JSON.stringify({ slot }) }),
  loadSlot: (slot) =>
    request('/api/saves/load', { method: 'POST', body: JSON.stringify({ slot }) }),
  gallery: () => request('/api/gallery'),
  store: () => request('/api/store'),
  purchase: (chapterId) =>
    request('/api/store/purchase', {
      method: 'POST',
      body: JSON.stringify({ chapterId }),
    }),
  chapters: () => request('/api/chapters'),
  chapterMap: (id) => request(`/api/chapters/${id}/map`),
};
