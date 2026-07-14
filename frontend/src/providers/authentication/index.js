// AuthenticationProvider — DEV STUB (bọc dưới NotificationProvider, trên GameProvider).
// Backend tự tạo phiên ẩn danh (cookie HttpOnly `uid`) ở lần gọi API đầu tiên, nên client
// KHÔNG có bước đăng nhập và KHÔNG đọc/xoá được cookie. Ta mô hình hoá một phiên ẩn danh
// lạc quan. Production (xem CLAUDE.md "Dev stubs to replace"): gọi GET /api/me khi mount,
// lưu currentUser thật, nối signIn/signOut vào session/JWT (NextAuth/Firebase) + /api/logout.
import { useCallback, useEffect, useMemo, useState } from 'react';

// Contexts
import { AuthenticationContext } from '@contexts/authentication';
import { useNotificationContext } from '@contexts/notification';

// Hooks
import useTranslation from '@hooks/useTranslation';

const ANON_USER = { id: 'anonymous', anonymous: true };

const AuthenticationProvider = ({ children }) => {
  const { t } = useTranslation();
  const { notify } = useNotificationContext();
  const [currentUser, setCurrentUser] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  // "Bootstrap" phiên. Dev không có /api/me; cookie phiên được server tạo lazy ở
  // request có auth đầu tiên → ta coi như đã có phiên ẩn danh.
  useEffect(() => {
    setCurrentUser(ANON_USER);
    setIsLoading(false);
  }, []);

  const signOut = useCallback(() => {
    // Cookie `uid` là HttpOnly → client không xoá được; cần /api/logout ở production.
    notify({
      kind: 'auth',
      title: t('auth_w1_signout_title', 'Phiên ẩn danh (dev)'),
      body: t('auth_w1_signout_body', 'Đăng xuất sẽ nối khi có session/JWT thật.'),
    });
  }, [notify, t]);

  return (
    <AuthenticationContext.Provider
      value={useMemo(
        () => ({ currentUser, isAuthenticated: currentUser != null, isLoading, signOut }),
        [currentUser, isLoading, signOut],
      )}
    >
      {children}
    </AuthenticationContext.Provider>
  );
};

export default AuthenticationProvider;
