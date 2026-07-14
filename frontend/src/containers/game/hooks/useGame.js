// Vòng lặp chơi (state + nghiệp vụ) — tách khỏi UI theo container pattern.
// current → phát clip → (linear: advance | choice: advance(choiceId)) → cập nhật HUD
// từ state SERVER trả về → lặp; 402 CHAPTER_LOCKED → mở StoreScreen.
import { useCallback, useEffect, useState } from 'react';

// Models
import { api, ApiError } from '@models/game';

// Constants
import { TOAST_TTL_MS } from '../constant';

export const useGame = () => {
  // State
  const [scene, setScene] = useState(null);
  const [started, setStarted] = useState(false);
  const [busy, setBusy] = useState(false);
  const [panel, setPanel] = useState('none');
  const [screen, setScreen] = useState('title');
  const [selectedChapter, setSelectedChapter] = useState(0);
  const [lockedChapterId, setLockedChapterId] = useState(undefined);
  const [error, setError] = useState('');

  // Callbacks
  const handleError = useCallback((e) => {
    if (e instanceof ApiError && e.code === 'CHAPTER_LOCKED') {
      setLockedChapterId(e.data?.chapterId);
      setPanel('store');
      return;
    }
    setError(e instanceof Error ? e.message : String(e));
  }, []);

  const start = useCallback(() => {
    setStarted(true);
    api.current().then(setScene).catch(handleError);
  }, [handleError]);

  // Bấm 1 node video trên bản đồ → nhảy tới đúng scene đó rồi vào player.
  const playScene = useCallback(
    (code) => {
      if (!code) return start(); // node chưa gắn scene → chơi tiếp từ save
      setStarted(true);
      api.jumpScene(code).then(setScene).catch(handleError);
    },
    [handleError, start],
  );

  const advance = useCallback(
    (choiceId) => {
      setBusy(true);
      api
        .advance(choiceId)
        .then(setScene)
        .catch(handleError)
        .finally(() => setBusy(false));
    },
    [handleError],
  );

  const restart = useCallback(() => {
    setBusy(true);
    api
      .restart()
      .then(setScene)
      .catch(handleError)
      .finally(() => setBusy(false));
  }, [handleError]);

  // Đóng video, quay về màn bản đồ chapter (giữ chapter đang chọn).
  const exitPlayer = useCallback(() => {
    setBusy(false);
    setStarted(false);
    setScreen('chapterMap');
  }, []);

  // Sau khi mua chương: thử advance lại để đi tiếp qua ranh giới chương.
  const resumeAfterPurchase = useCallback(() => {
    setPanel('none');
    setLockedChapterId(undefined);
    advance();
  }, [advance]);

  // Effects — toast tự ẩn sau TOAST_TTL_MS, hoặc khi user click/chạm ra chỗ khác.
  useEffect(() => {
    if (!error) return;
    const timer = setTimeout(() => setError(''), TOAST_TTL_MS);
    // Đợi 1 tick rồi mới gắn listener để không dính chính cú click vừa mở thông báo.
    let dismiss;
    const arm = setTimeout(() => {
      dismiss = () => setError('');
      window.addEventListener('pointerdown', dismiss);
    }, 0);
    return () => {
      clearTimeout(timer);
      clearTimeout(arm);
      if (dismiss) window.removeEventListener('pointerdown', dismiss);
    };
  }, [error]);

  return {
    scene,
    started,
    busy,
    panel,
    screen,
    selectedChapter,
    lockedChapterId,
    error,
    setScene,
    setPanel,
    setScreen,
    setSelectedChapter,
    setError,
    start,
    playScene,
    advance,
    restart,
    exitPlayer,
    resumeAfterPurchase,
  };
};
