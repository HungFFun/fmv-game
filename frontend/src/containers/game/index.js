// <GameShell> — lớp trình bày của feature "game": render theo state từ context,
// không tự suy ra game state (thin-client). Logic nằm ở provider/hooks.
import { useEffect, useState } from 'react';

// MUI
import Box from '@mui/material/Box';

// Context
import { useGameContext } from './context';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Components
import PlayerCore from './components/playerCore';
import TitleScreen from './components/titleScreen';
import ChapterSelectScreen from './components/chapterSelectScreen';
import ChapterMapScreen from './components/chapterMapScreen';
import AlbumScreen from './components/albumScreen';
import ChapterCompleteScreen from './components/chapterCompleteScreen';
import { SaveLoadMenu, StoreScreen } from './components/panels';

// Banner "demo không hỗ trợ" — Figma node 191:1689: dải gradient ngang (xanh→hồng→xanh,
// mờ dần 2 mép), chữ trắng căn giữa. Kích thước bám bề rộng stage 16:9 (dùng vw/vh vì banner
// nằm NGOÀI container stage nên không dùng được cqw).
const STAGE_W = 'min(100cqw, calc(100cqh * 16 / 9))';
const TOAST_SX = {
  position: 'absolute',
  top: '50%',
  left: '50%',
  transform: 'translate(-50%, -50%)',
  width: STAGE_W,
  textAlign: 'center',
  color: '#fff',
  fontFamily: "'Roboto', sans-serif",
  fontWeight: 500,
  fontSize: `calc(${STAGE_W} * 0.0219)`,
  lineHeight: 1.1,
  py: `calc(${STAGE_W} * 0.022)`,
  whiteSpace: 'nowrap',
  background:
    'linear-gradient(90deg, rgba(31,168,253,0) 0%, rgb(31,168,253) 12.96%, rgb(175,71,172) 42.31%, rgb(175,71,172) 60.1%, rgb(31,168,253) 87.81%, rgba(31,168,253,0) 100%)',
  pointerEvents: 'none',
  zIndex: 40,
};

const GameShell = () => {
  const { t } = useTranslation();
  const {
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
    playScene,
    advance,
    restart,
    exitPlayer,
    resumeAfterPurchase,
  } = useGameContext();

  // Màn Chapter Complete chỉ hiện SAU khi clip ending phát xong (PlayerCore báo lên).
  // Reset khi đổi scene (vd sau khi Chơi lại → scene mới).
  const [chapterDone, setChapterDone] = useState(false);
  const sceneId = scene?.scene?.id;
  useEffect(() => setChapterDone(false), [sceneId]);

  // Điều hướng trước khi vào game (title → chọn chapter → bản đồ chapter).
  if (!started) {
    return (
      <>
        {screen === 'title' && (
          <TitleScreen
            onStart={() => setScreen('chapterSelect')}
            onOpenGallery={() => setPanel('gallery')}
            onComingSoon={() => setError(t('game_w1_demo_unsupported', 'This feature is not supported by the demo version!'))}
          />
        )}
        {screen === 'chapterSelect' && (
          <ChapterSelectScreen
            onBack={() => setScreen('title')}
            onOpenChapter={(id) => {
              setSelectedChapter(id);
              setScreen('chapterMap');
            }}
            onLocked={() => setError(t('game_w1_demo_unsupported', 'This feature is not supported by the demo version!'))}
          />
        )}
        {screen === 'chapterMap' && (
          <ChapterMapScreen
            chapterId={selectedChapter}
            onBack={() => setScreen('chapterSelect')}
            onPlayVideo={(sceneCode) => playScene(sceneCode)}
          />
        )}
        {panel === 'gallery' && <AlbumScreen onClose={() => setPanel('none')} />}
        {error && <Box sx={TOAST_SX}>{error}</Box>}
      </>
    );
  }

  return (
    <Box sx={{ position: 'relative', height: '100%' }}>
      {scene && (
        <PlayerCore
          scene={scene}
          busy={busy}
          onLinearEnded={() => advance()}
          onEndingEnded={() => setChapterDone(true)}
          onChoose={(id) => advance(id)}
          onClose={exitPlayer}
        />
      )}

      {/* Chapter Complete — chỉ khi clip ending đã phát xong */}
      {scene && scene.scene.type === 'ending' && chapterDone && (
        <ChapterCompleteScreen
          ending={scene.ending}
          onRestart={restart}
          onGallery={() => setPanel('gallery')}
        />
      )}

      {panel === 'saves' && <SaveLoadMenu onLoaded={setScene} onClose={() => setPanel('none')} />}
      {panel === 'gallery' && <AlbumScreen onClose={() => setPanel('none')} />}
      {panel === 'store' && (
        <StoreScreen
          highlightChapterId={lockedChapterId}
          onPurchased={resumeAfterPurchase}
          onClose={() => setPanel('none')}
        />
      )}

      {error && <Box sx={TOAST_SX}>{error}</Box>}
    </Box>
  );
};

export default GameShell;
