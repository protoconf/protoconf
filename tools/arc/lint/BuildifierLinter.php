<?php

final class BuildifierLinter extends ArcanistExternalLinter {
  public function getInfoName() {
    return 'Buildifier';
  }

  public function getInfoURI() {
    return 'https://github.com/bazelbuild/buildtools/tree/master/buildifier';
  }

  public function getInfoDescription() {
    return pht('Formats Bazel BUILD files.');
  }
  public function getLinterName() {
    return 'BUILDIFIER';
  }

  public function getLinterConfigurationName() {
    return 'buildifier';
  }

  public function getDefaultBinary() {
    return __DIR__ . "/buildifier.sh";
  }

  public function getVersion() {
    return "unknown";
  }

  protected function getMandatoryFlags() {
    return array();
  }

  public function getInstallInstructions() {
    return pht(
      'You don\'t have buildifier in your PATH. Are you using bazel-dev container?',
      '',
      ''
    );
  }

  protected function parseLinterOutput($path, $err, $stdout, $stderr) {
    if ($err) {
      return false;
    }
    $messages = array();
    $original = $this->getData($path);
    if ($original !== $stdout) {
      $message = new ArcanistLintMessage();
      $message->setPath($path);
      $message->setSeverity(ArcanistLintSeverity::SEVERITY_AUTOFIX);
      $message->setName('Buildifier Format');
      $message->setCode($this->getLinterName());
      $message->setDescription('Wrongly formatted.');
      $message->setLine(1);
      $message->setChar(1);
      $message->setOriginalText($original);
      $message->setReplacementText($stdout);
      $messages[] = $message;
    }
    return $messages;
  }
}