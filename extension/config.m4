PHP_ARG_ENABLE([observer],
  [whether to enable observer support],
  [AS_HELP_STRING([--enable-observer],
    [Enable observer support])],
  [yes])

if test "$PHP_OBSERVER" != "no"; then
  AC_DEFINE(HAVE_OBSERVER, 1, [ Have observer support ])
  PHP_NEW_EXTENSION(observer, observer.c, $ext_shared)
fi