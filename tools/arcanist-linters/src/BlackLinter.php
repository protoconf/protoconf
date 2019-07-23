<?php
/**
 * Copyright 2019 Pinterest, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Formats Python files using Black
 */
final class BlackLinter extends ArcanistExternalLinter {
  private $linelen = null;

  public function getInfoName() {
    return 'Black';
  }

  public function getInfoURI() {
    return 'https://black.readthedocs.io/';
  }

  public function getInfoDescription() {
    return pht('Black is an opionionated code formatter for Python');
  }

  public function getLinterName() {
    return 'BLACK';
  }

  public function getLinterConfigurationName() {
    return 'black';
  }

  public function getLinterPriority() {
    return 0.01;
  }
  public function getDefaultBinary() {
    return 'black';
  }

  public function getVersion() {
    list($err, $stdout, $stderr) = exec_manual('%C --version', $this->getExecutableCommand());
    return trim(str_replace('black, version', '', $stdout));
  }

  public function getLinterConfigurationOptions() {
    $options = array(
      'black.linelen' => array(
        'type' => 'optional int',
        'help' => pht('Specify a non-default line length for black'),
      ),
    );
    return $options + parent::getLinterConfigurationOptions();
  }

  public function setLinterConfigurationValue($key, $value) {
    switch ($key) {
      case 'black.linelen':
        $this->linelen = $value;
        return;
    }
    return parent::setLinterConfigurationValue($key, $value);
  }

  protected function getMandatoryFlags() {
    $flags = array('--quiet');
    if ($this->linelen != null) {
        array_push($flags, '--line-length', $this->linelen);
    }
    return $flags;
  }

  public function getInstallInstructions() {
    return pht('pip3 install black');
  }

  protected function parseLinterOutput($path, $err, $stdout, $stderr) {
    if ($err == 123 or $stderr) {
      return false;
    } else {
      return array();
    }
  }
}
