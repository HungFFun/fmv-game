// ProviderComposer — gộp nhiều provider thành cây lồng nhau, tránh "provider pyramid".
// Phần tử ĐẦU mảng = lớp NGOÀI CÙNG. (Giống ProviderComposer của _app.js bên Tevi.)
import { cloneElement } from 'react';

const ProviderComposer = ({ providers, children }) =>
  providers.reduceRight((acc, provider) => cloneElement(provider, undefined, acc), children);

export default ProviderComposer;
