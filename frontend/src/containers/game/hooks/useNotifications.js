// Đẩy notifications do SERVER phát (mỗi SceneResponse) lên NotificationProvider toàn cục.
// Guard theo identity của scene để StrictMode (effect chạy 2 lần ở dev) không nhân đôi toast.
import { useEffect, useRef } from 'react';

// Contexts
import { useNotificationContext } from '@contexts/notification';

export const useNotifications = (scene) => {
  const { notify } = useNotificationContext();
  const lastScene = useRef(null);

  useEffect(() => {
    if (!scene || scene === lastScene.current) return;
    lastScene.current = scene;
    // Bỏ toast mở khoá ảnh (kind 'gallery') — không hiển thị cho người chơi.
    scene.notifications?.filter((n) => n.kind !== 'gallery').forEach(notify);
  }, [scene, notify]);
};
