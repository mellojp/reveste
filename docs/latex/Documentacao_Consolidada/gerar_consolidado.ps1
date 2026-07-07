$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
$out = Join-Path $PSScriptRoot "main.tex"

function Read-Utf8([string]$path) {
    return [System.IO.File]::ReadAllText((Join-Path $root $path), [System.Text.Encoding]::UTF8)
}

function BodyFrom([string]$text, [string]$startPattern) {
    $start = [regex]::Match($text, $startPattern)
    if (-not $start.Success) {
        throw "Padrao inicial nao encontrado: $startPattern"
    }

    $body = $text.Substring($start.Index)
    $body = [regex]::Replace($body, "\\end\{document\}\s*$", "")
    return $body.Trim()
}

$requisitos = Read-Utf8 "docs\latex\Requisitos_SWE\main.tex"
$casos = Read-Utf8 "docs\latex\Casos_de_Uso\main.tex"
$arquitetura = Read-Utf8 "docs\latex\AquiteturaDeSoftwareES\main.tex"

$requisitosCorpo = BodyFrom $requisitos "\\section\{Introdu.+?\}"
$requisitosCorpo = [regex]::Replace($requisitosCorpo, "\\clearpage\s*\\section\{Ap.+?ndice\}.*", "", [System.Text.RegularExpressions.RegexOptions]::Singleline).Trim()
$requisitosCorpo = [regex]::Replace(
    $requisitosCorpo,
    "As restri.+?es gerais do sistema s.+?o divididas sobre as seguintes categorias: Tecnol.+?gica, Operacional, Legal e de Integra.+?o\.\s*A tabela com a descri.+?o das restri.+?es e suas respectivas categorias encontra-se listada no Ap.+?ndice\.",
    "As restri\c{c}\~oes gerais do sistema s\~ao divididas sobre as seguintes categorias: Tecnol\'ogica, Operacional, Legal e de Integra\c{c}\~ao.",
    [System.Text.RegularExpressions.RegexOptions]::Singleline
)

$casosCorpo = BodyFrom $casos "\\section\{Introdu.+?\}"

$arquiteturaCorpo = BodyFrom $arquitetura "\\section\{Identifica.+?o da Architecture Description\}"
$arquiteturaCorpo = $arquiteturaCorpo.Replace("{diagrama_camadas.png}", "{../AquiteturaDeSoftwareES/diagrama_camadas.png}")
$arquiteturaCorpo = $arquiteturaCorpo.Replace("{diagrama_implantacao.png}", "{../AquiteturaDeSoftwareES/diagrama_implantacao.png}")
$arquiteturaCorpo = $arquiteturaCorpo.Replace("{modelo_conceitual_dominio_vp1.drawio.png}", "{../AquiteturaDeSoftwareES/modelo_conceitual_dominio_vp1.drawio.png}")

$diagramasDir = Join-Path $root "docs\diagramas"
function DiagramPath([string]$prefix) {
    $file = Get-ChildItem -LiteralPath $diagramasDir -File | Where-Object { $_.Name -like "$prefix*" } | Select-Object -First 1
    if ($null -eq $file) {
        throw "Diagrama nao encontrado com prefixo: $prefix"
    }
    return "../../diagramas/" + $file.Name.Replace("\", "/")
}

$diagramaClasses = DiagramPath "diagrama de classes"
$diagramaCarrinho = DiagramPath "diagrama de sequ* - adicionar"
$diagramaComprar = DiagramPath "diagrama de sequ* - comprar"
$diagramaPublicar = DiagramPath "diagrama de sequ* - publicar"

$content = @"
\documentclass[12pt,a4paper]{article}

\usepackage[utf8]{inputenc}
\usepackage[T1]{fontenc}
\usepackage[brazilian]{babel}
\usepackage{geometry}
\usepackage{graphicx}
\usepackage{hyperref}
\usepackage{tabularx}
\usepackage{booktabs}
\usepackage{enumitem}
\usepackage{indentfirst}
\usepackage{titlesec}
\usepackage{fancyhdr}
\usepackage{longtable}
\usepackage{array}
\usepackage{float}
\usepackage{adjustbox}
\usepackage[table]{xcolor}
\usepackage{microtype}

\geometry{left=3cm,right=2cm,top=3cm,bottom=2cm}
\setlength{\parindent}{1.3cm}
\setlength{\parskip}{0.2cm}

\hypersetup{
    colorlinks=true,
    linkcolor=black,
    urlcolor=blue,
    citecolor=black,
    pdftitle={Documentacao Consolidada - Sistema de Intermediacao de Brechos},
    pdfauthor={Caroline Santos, Beatriz Eduao, Ian Daniel, Joao Pedro M., Arthur Soares, Luiz Manoel}
}

\title{\textbf{Documenta\c{c}\~ao Consolidada do\\Sistema de Intermedia\c{c}\~ao de Brech\'os}}
\author{Caroline Santos, Beatriz Eduao, Ian Daniel\\Joao Pedro M., Arthur Soares, Luiz Manoel}
\date{Aracaju -- SE\\2026}

\begin{document}

\begin{titlepage}
\centering

\textbf{UNIVERSIDADE FEDERAL DE SERGIPE (UFS)}\\
\textbf{CENTRO DE CI\^ENCIAS EXATAS E TECNOLOGIA (CCET)}\\
\textbf{DEPARTAMENTO DE COMPUTA\c{C}\~AO (DCOMP)}\\
\textbf{ENGENHARIA DE SOFTWARE -- COMP0503}\\

\vspace{3cm}

{\LARGE \textbf{Documenta\c{c}\~ao Consolidada do\\Sistema de Intermedia\c{c}\~ao de Brech\'os}}\\

\vspace{3cm}

Caroline Santos, Beatriz Eduao, Ian Daniel\\
Joao Pedro M., Arthur Soares, Luiz Manoel\\

\vspace{2cm}

Prof. Dr. Michel Soares\\

\vfill

Aracaju -- SE\\
2026

\end{titlepage}

\tableofcontents
\clearpage

\part{Requisitos e An\'alise em Engenharia de Software}
$requisitosCorpo

\clearpage
\part{Especifica\c{c}\~ao dos Casos de Uso}
$casosCorpo

\clearpage
\part{Diagramas}
\label{part:diagramas}

\begin{figure}[H]
\centering
\includegraphics[width=\textwidth,height=0.85\textheight,keepaspectratio]{\detokenize{$diagramaClasses}}
\caption{Diagrama de classes em nivel de projeto}
\end{figure}

\clearpage
\begin{figure}[H]
\centering
\includegraphics[width=\textwidth,height=0.85\textheight,keepaspectratio]{\detokenize{$diagramaCarrinho}}
\caption{Diagrama de sequencia: adicionar ao carrinho}
\end{figure}

\clearpage
\begin{figure}[H]
\centering
\includegraphics[width=\textwidth,height=0.85\textheight,keepaspectratio]{\detokenize{$diagramaComprar}}
\caption{Diagrama de sequencia: comprar item}
\end{figure}

\clearpage
\begin{figure}[H]
\centering
\includegraphics[width=\textwidth,height=0.85\textheight,keepaspectratio]{\detokenize{$diagramaPublicar}}
\caption{Diagrama de sequencia: publicar anuncio}
\end{figure}

\clearpage
\part{Descri\c{c}\~ao Arquitetural}
$arquiteturaCorpo

\end{document}
"@

[System.IO.File]::WriteAllText($out, $content, [System.Text.Encoding]::UTF8)
