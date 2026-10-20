import type { ImgHTMLAttributes } from 'react';
import { useTranslation } from '@rovo/i18n';
import { cn } from '@rovo/ui';

export interface BrandLogoProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'src' | 'alt'> {
  /** Height in px (width auto). */
  height?: number;
}

/**
 * The per-app header logo from `public/logo-header.{webp,png}` (generated from the approved
 * brand marks, ≤ 20 KB). WebP with PNG fallback; never the ~1 MB raster logo.
 */
export function BrandLogo({ height = 28, className, ...rest }: BrandLogoProps) {
  const { t } = useTranslation();
  return (
    <picture>
      <source srcSet="/logo-header.webp" type="image/webp" />
      <img
        src="/logo-header.png"
        alt={t('brand.logoAlt')}
        height={height}
        className={cn('w-auto select-none', className)}
        style={{ height }}
        {...rest}
      />
    </picture>
  );
}
