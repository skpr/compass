<?php

use Drupal\Core\Config\BootstrapConfigStorageFactory;
use Drupal\Core\Database\Database;

$settings['container_yamls'][] = __DIR__ . '/services.yml';

$settings['allow_authorize_operations'] = FALSE;

$config['cron_safe_threshold'] = '0';
$settings['file_public_path'] = 'sites/default/files';
$config['system.file']['path']['temporary'] = 'sites/default/files/tmp';
$settings['file_private_path'] = 'sites/default/files/private';

$settings['hash_salt'] = !empty($settings['hash_salt']) ? $settings['hash_salt'] : 'xxxxxxxxxxxxxxxxxxxx';

$settings['trusted_host_patterns'][] = '^127\.0\.0\.1$';

$settings['php_storage']['twig'] = [
  'directory' => (DRUPAL_ROOT . '/..') . '/.php',
];

$settings['config_sync_directory'] = DRUPAL_ROOT . '/../config-export';

$settings['deployment_identifier'] = getenv('SKPR_VERSION') ?? \Drupal::VERSION;
