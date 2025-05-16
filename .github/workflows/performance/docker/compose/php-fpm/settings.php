<?php

use Drupal\Core\Config\BootstrapConfigStorageFactory;
use Drupal\Core\Database\Database;

$settings['container_yamls'][] = __DIR__ . '/services.yml';

$settings['allow_authorize_operations'] = FALSE;

$databases['default']['default'] = array(
  'driver' => 'mysql',
  'database' => 'local',
  'username' => 'local',
  'password' => 'local',
  'host' => '127.0.0.1',
);

$config['cron_safe_threshold'] = '0';
$settings['file_public_path'] = 'sites/default/files';
$config['system.file']['path']['temporary'] = '/mnt/temporary';
$settings['file_private_path'] = '/mnt/private';

$settings['hash_salt'] = !empty($settings['hash_salt']) ? $settings['hash_salt'] : 'xxxxxxxxxxxxxxxxxxxx';

$settings['trusted_host_patterns'][] = '^127\.0\.0\.1$';

$settings['php_storage']['twig'] = [
  'directory' => (DRUPAL_ROOT . '/..') . '/.php',
];

$settings['config_sync_directory'] = DRUPAL_ROOT . '/../config-export';

$settings['deployment_identifier'] = getenv('SKPR_VERSION') ?? \Drupal::VERSION;

$settings['cache']['bins']['render'] = 'cache.backend.null';
$settings['cache']['bins']['page'] = 'cache.backend.null';
$settings['cache']['bins']['dynamic_page_cache'] = 'cache.backend.null';
